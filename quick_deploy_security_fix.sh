#!/bin/bash

# NPS Security Fix Quick Deploy Script
# This script automates the security patch deployment process

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
print_header() {
    echo -e "\n${BLUE}=========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}=========================================${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
    exit 1
}

print_header "NPS Security Fix Deployment"

# Check if running with sufficient privileges
if [ "$EUID" -eq 0 ]; then 
    print_warning "Running as root. This is not recommended."
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Step 1: Backup
print_header "Step 1: Creating Backups"

BACKUP_DIR="backup_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

if [ -d "conf" ]; then
    cp -r conf "$BACKUP_DIR/"
    print_success "Configuration backed up to $BACKUP_DIR/conf"
else
    print_warning "Configuration directory not found, skipping backup"
fi

if [ -f "nps" ]; then
    cp nps "$BACKUP_DIR/nps.backup"
    print_success "NPS binary backed up"
fi

# Step 2: Verify dependencies
print_header "Step 2: Checking Dependencies"

if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.15 or higher."
fi

GO_VERSION=$(go version | awk '{print $3}')
print_success "Go version: $GO_VERSION"

# Step 3: Download dependencies
print_header "Step 3: Installing Dependencies"

if go get golang.org/x/crypto/bcrypt; then
    print_success "bcrypt dependency installed"
else
    print_error "Failed to install bcrypt dependency"
fi

# Step 4: Build new version
print_header "Step 4: Building NPS with Security Patches"

if go build -o nps; then
    print_success "NPS built successfully"
else
    print_error "Build failed"
fi

# Step 5: Build migration tool
print_header "Step 5: Building Password Migration Tool"

if [ -f "tools/migrate_passwords.go" ]; then
    if go build -o tools/migrate_passwords tools/migrate_passwords.go; then
        print_success "Migration tool built successfully"
    else
        print_error "Failed to build migration tool"
    fi
else
    print_warning "Migration tool source not found"
fi

# Step 6: Run tests
print_header "Step 6: Running Security Tests"

if [ -f "test_security_fixes.sh" ]; then
    if bash test_security_fixes.sh; then
        print_success "All security tests passed"
    else
        print_error "Security tests failed. Please review the output."
    fi
else
    print_warning "Test script not found, skipping tests"
fi

# Step 7: Stop existing service
print_header "Step 7: Stopping Existing Service"

if pgrep -x "nps" > /dev/null; then
    print_warning "NPS is currently running"
    read -p "Stop NPS service? (Y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
        if ./nps stop 2>/dev/null || killall nps 2>/dev/null; then
            print_success "NPS service stopped"
            sleep 2
        else
            print_warning "Could not stop NPS automatically. Please stop it manually."
            read -p "Press Enter when ready to continue..."
        fi
    fi
else
    print_success "NPS is not running"
fi

# Step 8: Migrate passwords
print_header "Step 8: Migrating Passwords"

if [ -f "tools/migrate_passwords" ] && [ -f "conf/clients.json" ]; then
    read -p "Run password migration? This will convert plaintext passwords to bcrypt. (Y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
        if ./tools/migrate_passwords conf/clients.json; then
            print_success "Password migration completed"
        else
            print_error "Password migration failed"
        fi
    else
        print_warning "Password migration skipped - passwords will still work but won't be hashed"
    fi
else
    print_warning "Migration tool or clients.json not found, skipping password migration"
fi

# Step 9: Start service
print_header "Step 9: Starting NPS Service"

read -p "Start NPS now? (Y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
    if ./nps start 2>/dev/null || nohup ./nps &>/dev/null &; then
        sleep 2
        if pgrep -x "nps" > /dev/null; then
            print_success "NPS service started successfully"
        else
            print_warning "NPS may not have started correctly. Please check manually."
        fi
    else
        print_warning "Could not start NPS automatically. Please start it manually."
    fi
else
    print_warning "NPS not started. Start it manually when ready."
fi

# Step 10: Summary
print_header "Deployment Summary"

echo "Backup location: $BACKUP_DIR"
echo ""
echo "Security fixes applied:"
echo "  ✓ Password hashing with bcrypt"
echo "  ✓ AES encryption with random IV"
echo "  ✓ Constant-time password comparison"
echo "  ✓ Reduced authentication time window (5s)"
echo "  ✓ Session timeout (2 hours)"
echo "  ✓ Login brute force protection (3 attempts)"
echo "  ✓ Path traversal prevention"
echo "  ✓ CSRF protection"
echo "  ✓ Session regeneration on login"
echo ""

print_header "Post-Deployment Actions Required"

echo "1. Test the web interface:"
echo "   - Try logging in with existing credentials"
echo "   - Verify CSRF tokens in forms"
echo "   - Test session timeout (2 hours)"
echo ""

echo "2. Update all users to change their passwords:"
echo "   - Passwords are now required to be 8+ characters"
echo "   - Old passwords still work (backward compatible)"
echo "   - New passwords will be bcrypt hashed"
echo ""

echo "3. Review security documentation:"
echo "   - SECURITY_AUDIT_REPORT.md - Complete security audit"
echo "   - SECURITY_FIX_GUIDE.md - Detailed deployment guide"
echo "   - CHANGELOG_SECURITY.md - All changes made"
echo ""

echo "4. Consider additional security measures:"
echo "   - Enable HTTPS"
echo "   - Configure firewall rules"
echo "   - Set up regular backups"
echo "   - Enable detailed logging"
echo ""

if [ -d "$BACKUP_DIR" ]; then
    print_warning "Keep the backup directory for rollback: $BACKUP_DIR"
fi

print_header "Deployment Complete!"

echo -e "${GREEN}NPS security patches have been successfully deployed.${NC}"
echo ""
echo "If you encounter any issues, refer to SECURITY_FIX_GUIDE.md"
echo "or restore from backup: $BACKUP_DIR"
echo ""
