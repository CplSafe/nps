package controllers

import (
	"math/rand"
	"net"
	"sync"
	"time"

	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/file"
	"ehang.io/nps/server"
	"github.com/astaxie/beego"
)

type LoginController struct {
	beego.Controller
}

var ipRecord sync.Map

type record struct {
	hasLoginFailTimes int
	lastLoginTime     time.Time
}

func (self *LoginController) Index() {
	// Try login implicitly, will succeed if it's configured as no-auth(empty username&password).
	webBaseUrl := beego.AppConfig.String("web_base_url")
	if self.doLogin("", "", false) {
		self.Redirect(webBaseUrl+"/index/index", 302)
	}
	self.Data["web_base_url"] = webBaseUrl
	self.Data["register_allow"], _ = beego.AppConfig.Bool("allow_user_register")
	self.TplName = "login/index.html"
}

func (self *LoginController) Verify() {
	username := self.GetString("username")
	password := self.GetString("password")
	if self.doLogin(username, password, true) {
		self.Data["json"] = map[string]interface{}{"status": 1, "msg": "login success"}
	} else {
		self.Data["json"] = map[string]interface{}{"status": 0, "msg": "username or password incorrect"}
	}
	self.ServeJSON()
}

func (self *LoginController) doLogin(username, password string, explicit bool) bool {
	clearIprecord()
	ip, _, _ := net.SplitHostPort(self.Ctx.Request.RemoteAddr)
	
	// SECURITY FIX: Stricter brute force protection - 3 attempts, 30 minutes lockout
	if v, ok := ipRecord.Load(ip); ok {
		vv := v.(*record)
		// Reset after 30 minutes instead of 60 seconds
		if (time.Now().Unix() - vv.lastLoginTime.Unix()) >= 1800 {
			vv.hasLoginFailTimes = 0
		}
		// Lock after 3 failed attempts instead of 10
		if vv.hasLoginFailTimes >= 3 {
			return false
		}
	}
	
	var auth bool
	configPassword := beego.AppConfig.String("web_password")
	configUsername := beego.AppConfig.String("web_username")
	
	// SECURITY FIX: Use constant-time comparison for password check
	if crypt.SecureCompare(username, configUsername) && crypt.SecureCompare(password, configPassword) {
		// Regenerate session ID to prevent session fixation (SECURITY FIX)
		self.DestroySession()
		self.StartSession()
		
		self.SetSession("isAdmin", true)
		self.DelSession("clientId")
		self.DelSession("username")
		// Add session timeout (SECURITY FIX)
		self.SetSession("loginTime", time.Now().Unix())
		auth = true
		server.Bridge.Register.Store(common.GetIpByAddr(self.Ctx.Input.IP()), time.Now().Add(time.Hour*time.Duration(2)))
	}
	
	b, err := beego.AppConfig.Bool("allow_user_login")
	if err == nil && b && !auth {
		file.GetDb().JsonDb.Clients.Range(func(key, value interface{}) bool {
			v := value.(*file.Client)
			if !v.Status || v.NoDisplay {
				return true
			}
			
			// Check if password is in new bcrypt format or old plaintext
			var passwordMatch bool
			if v.WebPassword != "" && len(v.WebPassword) == 60 && v.WebPassword[:4] == "$2a$" {
				// New bcrypt format (SECURITY FIX)
				passwordMatch = crypt.CheckPasswordHash(password, v.WebPassword)
			} else {
				// Legacy plaintext format - use constant-time compare
				passwordMatch = crypt.SecureCompare(password, v.WebPassword)
			}
			
			if v.WebUserName == "" && v.WebPassword == "" {
				if crypt.SecureCompare(username, "user") && crypt.SecureCompare(v.VerifyKey, password) {
					auth = true
				} else {
					return true
				}
			}
			
			if !auth && passwordMatch && crypt.SecureCompare(v.WebUserName, username) {
				auth = true
			}
			
			if auth {
				// Regenerate session ID (SECURITY FIX)
				self.DestroySession()
				self.StartSession()
				
				self.SetSession("isAdmin", false)
				self.SetSession("clientId", v.Id)
				self.SetSession("username", v.WebUserName)
				self.SetSession("loginTime", time.Now().Unix())
				return false
			}
			return true
		})
	}
	
	if auth {
		self.SetSession("auth", true)
		ipRecord.Delete(ip)
		return true
	}
	
	if v, load := ipRecord.LoadOrStore(ip, &record{hasLoginFailTimes: 1, lastLoginTime: time.Now()}); load && explicit {
		vv := v.(*record)
		vv.lastLoginTime = time.Now()
		vv.hasLoginFailTimes += 1
		ipRecord.Store(ip, vv)
	}
	return false
}
func (self *LoginController) Register() {
	if self.Ctx.Request.Method == "GET" {
		self.Data["web_base_url"] = beego.AppConfig.String("web_base_url")
		self.TplName = "login/register.html"
	} else {
		if b, err := beego.AppConfig.Bool("allow_user_register"); err != nil || !b {
			self.Data["json"] = map[string]interface{}{"status": 0, "msg": "register is not allow"}
			self.ServeJSON()
			return
		}
		username := self.GetString("username")
		password := self.GetString("password")
		
		// SECURITY FIX: Enhanced input validation
		if username == "" || password == "" || username == beego.AppConfig.String("web_username") {
			self.Data["json"] = map[string]interface{}{"status": 0, "msg": "please check your input"}
			self.ServeJSON()
			return
		}
		
		// SECURITY FIX: Password strength requirement
		if len(password) < 8 {
			self.Data["json"] = map[string]interface{}{"status": 0, "msg": "password must be at least 8 characters"}
			self.ServeJSON()
			return
		}
		
		// SECURITY FIX: Hash password with bcrypt
		hashedPassword, err := crypt.HashPassword(password)
		if err != nil {
			self.Data["json"] = map[string]interface{}{"status": 0, "msg": "password hashing failed"}
			self.ServeJSON()
			return
		}
		
		t := &file.Client{
			Id:          int(file.GetDb().JsonDb.GetClientId()),
			Status:      true,
			Cnf:         &file.Config{},
			WebUserName: username,
			WebPassword: hashedPassword, // Store hashed password
			Flow:        &file.Flow{},
		}
		if err := file.GetDb().NewClient(t); err != nil {
			self.Data["json"] = map[string]interface{}{"status": 0, "msg": err.Error()}
		} else {
			self.Data["json"] = map[string]interface{}{"status": 1, "msg": "register success"}
		}
		self.ServeJSON()
	}
}

func (self *LoginController) Out() {
	self.SetSession("auth", false)
	self.Redirect(beego.AppConfig.String("web_base_url")+"/login/index", 302)
}

func clearIprecord() {
	rand.Seed(time.Now().UnixNano())
	x := rand.Intn(100)
	if x == 1 {
		ipRecord.Range(func(key, value interface{}) bool {
			v := value.(*record)
			if time.Now().Unix()-v.lastLoginTime.Unix() >= 60 {
				ipRecord.Delete(key)
			}
			return true
		})
	}
}
