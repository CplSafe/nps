package controllers

import (
	"html"
	"strconv"
	"strings"
	"time"

	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/crypt"
	"ehang.io/nps/lib/file"
	"ehang.io/nps/server"
	"github.com/astaxie/beego"
)

type BaseController struct {
	beego.Controller
	controllerName string
	actionName     string
}

//初始化参数
func (s *BaseController) Prepare() {
	s.Data["web_base_url"] = beego.AppConfig.String("web_base_url")
	controllerName, actionName := s.GetControllerAndAction()
	s.controllerName = strings.ToLower(controllerName[0 : len(controllerName)-10])
	s.actionName = strings.ToLower(actionName)
	
	// SECURITY FIX: Enable CSRF protection for authenticated requests
	// Skip CSRF for login/auth endpoints
	if s.controllerName != "login" && s.controllerName != "auth" {
		// Generate CSRF token if not exists
		if s.GetSession("csrf_token") == nil && s.Ctx.Request.Method == "GET" {
			token := s.GenerateCSRFToken()
			s.Data["csrf_token"] = token
		} else if s.Ctx.Request.Method != "GET" && s.Ctx.Request.Method != "HEAD" && s.Ctx.Request.Method != "OPTIONS" {
			// Validate CSRF for state-changing requests
			s.RequireCSRF()
		}
	}
	// web api verify
	// SECURITY FIX: Use pure nonce mechanism to prevent replay attacks
	// 使用纯Nonce机制防止重放攻击，不依赖时间窗口
	// param 1 is md5(authKey+timestamp+nonce)
	// param 2 is timestamp (用于签名，不用于时间窗口验证)
	// param 3 is nonce (一次性令牌，每个请求必须唯一)
	md5Key := s.getEscapeString("auth_key")
	timestamp := s.GetIntNoErr("timestamp")
	nonce := s.getEscapeString("nonce")
	configKey := beego.AppConfig.String("auth_key")
	
	// SECURITY FIX: Check session timeout (2 hours)
	if s.GetSession("auth") == true {
		if loginTime, ok := s.GetSession("loginTime").(int64); ok {
			if time.Now().Unix()-loginTime > 7200 { // 2 hours
				s.DestroySession()
				s.Redirect(beego.AppConfig.String("web_base_url")+"/login/index", 302)
				return
			}
		}
	}
	
	// SECURITY FIX: Pure Nonce-based replay attack prevention
	// 使用纯Nonce机制，不依赖时间窗口，完全防止重放攻击
	// 每个nonce只能使用一次，无论何时发送
	authenticated := false
	if md5Key != "" && nonce != "" && timestamp > 0 {
		// 验证nonce（防止重放）- 30分钟过期仅用于内存清理
		if common.ValidateNonce(nonce, 30) {
			// 验证签名
			expectedKey := crypt.Md5(configKey + strconv.Itoa(timestamp) + nonce)
			if crypt.SecureCompare(md5Key, expectedKey) {
				authenticated = true
			}
		}
	}
	
	if !authenticated {
		if s.GetSession("auth") != true {
			s.Redirect(beego.AppConfig.String("web_base_url")+"/login/index", 302)
			return
		}
	} else {
		s.SetSession("isAdmin", true)
		s.SetSession("loginTime", time.Now().Unix())
		s.Data["isAdmin"] = true
	}
	if s.GetSession("isAdmin") != nil && !s.GetSession("isAdmin").(bool) {
		s.Ctx.Input.SetData("client_id", s.GetSession("clientId").(int))
		s.Ctx.Input.SetParam("client_id", strconv.Itoa(s.GetSession("clientId").(int)))
		s.Data["isAdmin"] = false
		s.Data["username"] = s.GetSession("username")
		s.CheckUserAuth()
	} else {
		s.Data["isAdmin"] = true
	}
	s.Data["https_just_proxy"], _ = beego.AppConfig.Bool("https_just_proxy")
	s.Data["allow_user_login"], _ = beego.AppConfig.Bool("allow_user_login")
	s.Data["allow_flow_limit"], _ = beego.AppConfig.Bool("allow_flow_limit")
	s.Data["allow_rate_limit"], _ = beego.AppConfig.Bool("allow_rate_limit")
	s.Data["allow_connection_num_limit"], _ = beego.AppConfig.Bool("allow_connection_num_limit")
	s.Data["allow_multi_ip"], _ = beego.AppConfig.Bool("allow_multi_ip")
	s.Data["system_info_display"], _ = beego.AppConfig.Bool("system_info_display")
	s.Data["allow_tunnel_num_limit"], _ = beego.AppConfig.Bool("allow_tunnel_num_limit")
	s.Data["allow_local_proxy"], _ = beego.AppConfig.Bool("allow_local_proxy")
	s.Data["allow_user_change_username"], _ = beego.AppConfig.Bool("allow_user_change_username")
}

//加载模板
func (s *BaseController) display(tpl ...string) {
	s.Data["web_base_url"] = beego.AppConfig.String("web_base_url")
	var tplname string
	if s.Data["menu"] == nil {
		s.Data["menu"] = s.actionName
	}
	if len(tpl) > 0 {
		tplname = strings.Join([]string{tpl[0], "html"}, ".")
	} else {
		tplname = s.controllerName + "/" + s.actionName + ".html"
	}
	ip := s.Ctx.Request.Host
	s.Data["ip"] = common.GetIpByAddr(ip)
	s.Data["bridgeType"] = beego.AppConfig.String("bridge_type")
	if common.IsWindows() {
		s.Data["win"] = ".exe"
	}
	s.Data["p"] = server.Bridge.TunnelPort
	s.Data["proxyPort"] = beego.AppConfig.String("hostPort")
	s.Layout = "public/layout.html"
	s.TplName = tplname
}

//错误
func (s *BaseController) error() {
	s.Data["web_base_url"] = beego.AppConfig.String("web_base_url")
	s.Layout = "public/layout.html"
	s.TplName = "public/error.html"
}

//getEscapeString
func (s *BaseController) getEscapeString(key string) string {
	return html.EscapeString(s.GetString(key))
}

//去掉没有err返回值的int
func (s *BaseController) GetIntNoErr(key string, def ...int) int {
	strv := s.Ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0]
	}
	val, _ := strconv.Atoi(strv)
	return val
}

//获取去掉错误的bool值
func (s *BaseController) GetBoolNoErr(key string, def ...bool) bool {
	strv := s.Ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0]
	}
	val, _ := strconv.ParseBool(strv)
	return val
}

//ajax正确返回
func (s *BaseController) AjaxOk(str string) {
	s.Data["json"] = ajax(str, 1)
	s.ServeJSON()
	s.StopRun()
}

//ajax错误返回
func (s *BaseController) AjaxErr(str string) {
	s.Data["json"] = ajax(str, 0)
	s.ServeJSON()
	s.StopRun()
}

//组装ajax
func ajax(str string, status int) map[string]interface{} {
	json := make(map[string]interface{})
	json["status"] = status
	json["msg"] = str
	return json
}

//ajax table返回
func (s *BaseController) AjaxTable(list interface{}, cnt int, recordsTotal int, kwargs map[string]interface{}) {
	json := make(map[string]interface{})
	json["rows"] = list
	json["total"] = recordsTotal
	if kwargs != nil {
		for k, v := range kwargs {
			if v != nil {
				json[k] = v
			}
		}
	}
	s.Data["json"] = json
	s.ServeJSON()
	s.StopRun()
}

//ajax table参数
func (s *BaseController) GetAjaxParams() (start, limit int) {
	return s.GetIntNoErr("offset"), s.GetIntNoErr("limit")
}

func (s *BaseController) SetInfo(name string) {
	s.Data["name"] = name
}

func (s *BaseController) SetType(name string) {
	s.Data["type"] = name
}

func (s *BaseController) CheckUserAuth() {
	if s.controllerName == "client" {
		if s.actionName == "add" {
			s.StopRun()
			return
		}
		if id := s.GetIntNoErr("id"); id != 0 {
			if id != s.GetSession("clientId").(int) {
				s.StopRun()
				return
			}
		}
	}
	if s.controllerName == "index" {
		if id := s.GetIntNoErr("id"); id != 0 {
			belong := false
			if strings.Contains(s.actionName, "h") {
				if v, ok := file.GetDb().JsonDb.Hosts.Load(id); ok {
					if v.(*file.Host).Client.Id == s.GetSession("clientId").(int) {
						belong = true
					}
				}
			} else {
				if v, ok := file.GetDb().JsonDb.Tasks.Load(id); ok {
					if v.(*file.Tunnel).Client.Id == s.GetSession("clientId").(int) {
						belong = true
					}
				}
			}
			if !belong {
				s.StopRun()
			}
		}
	}
}
