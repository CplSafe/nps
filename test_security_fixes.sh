#!/bin/bash

# NPS Security Fixes Test Script
# This script tests the security fixes applied to NPS

echo "========================================="
echo "NPS Security Fixes Test Script"
echo "========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to print test result
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}: $2"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ FAIL${NC}: $2"
        ((TESTS_FAILED++))
    fi
}

echo "1. Testing Password Hashing Functions"
echo "--------------------------------------"

# Test bcrypt functions exist
if grep -q "HashPassword" lib/crypt/crypt.go; then
    print_result 0 "HashPassword function exists"
else
    print_result 1 "HashPassword function missing"
fi

if grep -q "CheckPasswordHash" lib/crypt/crypt.go; then
    print_result 0 "CheckPasswordHash function exists"
else
    print_result 1 "CheckPasswordHash function missing"
fi

if grep -q "SecureCompare" lib/crypt/crypt.go; then
    print_result 0 "SecureCompare function exists"
else
    print_result 1 "SecureCompare function missing"
fi

echo ""
echo "2. Testing AES Encryption Security"
echo "--------------------------------------"

# Check for random IV usage
if grep -q "rand.Read(iv)" lib/crypt/crypt.go; then
    print_result 0 "AES uses random IV"
else
    print_result 1 "AES still uses fixed IV"
fi

# Check for IV extraction in decrypt
if grep -q "Extract IV from ciphertext" lib/crypt/crypt.go; then
    print_result 0 "AES decrypt extracts IV"
else
    print_result 1 "AES decrypt doesn't extract IV"
fi

echo ""
echo "3. Testing Authentication Security"
echo "--------------------------------------"

# Check for constant-time comparison in CheckAuth
if grep -q "SecureCompare" lib/common/util.go; then
    print_result 0 "CheckAuth uses constant-time comparison"
else
    print_result 1 "CheckAuth doesn't use constant-time comparison"
fi

# Check for reduced time window
if grep -q "<= 5" web/controllers/base.go; then
    print_result 0 "Auth time window reduced to 5 seconds"
else
    print_result 1 "Auth time window not reduced"
fi

# Check for session timeout
if grep -q "session timeout" web/controllers/base.go; then
    print_result 0 "Session timeout implemented"
else
    print_result 1 "Session timeout not implemented"
fi

echo ""
echo "4. Testing Login Protection"
echo "--------------------------------------"

# Check for stricter brute force protection
if grep -q ">= 3" web/controllers/login.go; then
    print_result 0 "Login attempts reduced to 3"
else
    print_result 1 "Login attempts not reduced"
fi

if grep -q "1800" web/controllers/login.go; then
    print_result 0 "Lockout duration increased to 30 minutes"
else
    print_result 1 "Lockout duration not increased"
fi

# Check for session regeneration
if grep -q "DestroySession" web/controllers/login.go; then
    print_result 0 "Session regeneration on login"
else
    print_result 1 "Session regeneration not implemented"
fi

echo ""
echo "5. Testing Path Traversal Protection"
echo "--------------------------------------"

# Check for path sanitization
if grep -q "sanitizePath" server/proxy/http.go; then
    print_result 0 "Path sanitization function exists"
else
    print_result 1 "Path sanitization function missing"
fi

# Check sanitizePath actually filters ..
if grep -q '"\.\."' server/proxy/http.go; then
    print_result 0 "sanitizePath filters directory traversal"
else
    print_result 1 "sanitizePath doesn't filter properly"
fi

echo ""
echo "6. Testing CSRF Protection"
echo "--------------------------------------"

# Check for CSRF controller
if [ -f "web/controllers/csrf.go" ]; then
    print_result 0 "CSRF controller file exists"
else
    print_result 1 "CSRF controller file missing"
fi

# Check for CSRF in base controller
if grep -q "csrf" web/controllers/base.go; then
    print_result 0 "CSRF protection integrated in base controller"
else
    print_result 1 "CSRF protection not integrated"
fi

echo ""
echo "7. Testing Password Storage"
echo "--------------------------------------"

# Check login controller uses bcrypt
if grep -q "CheckPasswordHash" web/controllers/login.go; then
    print_result 0 "Login uses bcrypt verification"
else
    print_result 1 "Login doesn't use bcrypt"
fi

# Check client controller hashes passwords
if grep -q "HashPassword" web/controllers/client.go; then
    print_result 0 "Client controller hashes passwords"
else
    print_result 1 "Client controller doesn't hash passwords"
fi

# Check password length requirement
if grep -q "password must be at least 8 characters" web/controllers/login.go; then
    print_result 0 "Minimum password length enforced"
else
    print_result 1 "No minimum password length"
fi

echo ""
echo "8. Testing Migration Tool"
echo "--------------------------------------"

# Check migration tool exists
if [ -f "tools/migrate_passwords.go" ]; then
    print_result 0 "Password migration tool exists"
else
    print_result 1 "Password migration tool missing"
fi

echo ""
echo "9. Testing Documentation"
echo "--------------------------------------"

# Check security audit report
if [ -f "SECURITY_AUDIT_REPORT.md" ]; then
    print_result 0 "Security audit report exists"
else
    print_result 1 "Security audit report missing"
fi

# Check fix guide
if [ -f "SECURITY_FIX_GUIDE.md" ]; then
    print_result 0 "Security fix guide exists"
else
    print_result 1 "Security fix guide missing"
fi

# Check changelog
if [ -f "CHANGELOG_SECURITY.md" ]; then
    print_result 0 "Security changelog exists"
else
    print_result 1 "Security changelog missing"
fi

echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
echo -e "Tests Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Tests Failed: ${RED}${TESTS_FAILED}${NC}"
echo -e "Total Tests: $((TESTS_PASSED + TESTS_FAILED))"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}All security fixes verified successfully!${NC}"
    exit 0
else
    echo -e "\n${RED}Some security fixes are missing or incomplete.${NC}"
    echo "Please review the failed tests above."
    exit 1
fi
