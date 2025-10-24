#!/bin/bash

# API Endpoint Testing Script
# Usage: ./test_endpoints.sh [base_url]

BASE_URL="${1:-http://localhost:8080}"
PASS=0
FAIL=0
COOKIE_JAR="/tmp/test_cookies_$.txt"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Cleanup cookie file on exit
trap "rm -f $COOKIE_JAR" EXIT

# Function to test an endpoint (automatically uses cookies from jar)
test_endpoint() {
    local method=$1
    local endpoint=$2
    local expected_status=$3
    local data=$4
    local description=$5
    local use_auth=${6:-true}  # Default to true
    local verify_json=${7:-""}  # Optional: jq expression to verify response
    
    echo -e "\n${YELLOW}Testing:${NC} $description"
    echo "  ${method} ${BASE_URL}${endpoint}"
    
    # Build curl command based on whether we need auth
    if [ "$use_auth" = "true" ]; then
        # Use cookie jar for authenticated requests
        if [ -n "$data" ]; then
            response=$(curl -s -w "\n%{http_code}" -X "$method" \
                -H "Content-Type: application/json" \
                -b "$COOKIE_JAR" \
                -c "$COOKIE_JAR" \
                -d "$data" \
                "${BASE_URL}${endpoint}")
        else
            response=$(curl -s -w "\n%{http_code}" -X "$method" \
                -b "$COOKIE_JAR" \
                -c "$COOKIE_JAR" \
                "${BASE_URL}${endpoint}")
        fi
        echo "  [Using purch_token cookie from jar]"
    else
        # Don't send cookies for unauthenticated requests
        if [ -n "$data" ]; then
            response=$(curl -s -w "\n%{http_code}" -X "$method" \
                -H "Content-Type: application/json" \
                -d "$data" \
                "${BASE_URL}${endpoint}")
        else
            response=$(curl -s -w "\n%{http_code}" -X "$method" \
                "${BASE_URL}${endpoint}")
        fi
        echo "  [No authentication]"
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    # Check status code
    local status_pass=false
    if [ "$http_code" -eq "$expected_status" ]; then
        status_pass=true
    fi
    
    # Check JSON verification if provided
    local verify_pass=true
    local verify_msg=""
    if [ -n "$verify_json" ] && [ -n "$body" ]; then
        if command -v jq &> /dev/null; then
            verify_result=$(echo "$body" | jq -r "$verify_json" 2>/dev/null)
            if [ "$verify_result" = "true" ]; then
                verify_pass=true
                verify_msg=" | Response verified ✓"
            else
                verify_pass=false
                verify_msg=" | Verification failed: expected '$verify_json' to be true"
            fi
        else
            verify_msg=" | Warning: jq not installed, skipping verification"
        fi
    fi
    
    # Overall pass/fail
    if [ "$status_pass" = true ] && [ "$verify_pass" = true ]; then
        echo -e "  ${GREEN}✓ PASS${NC} (Status: $http_code)${verify_msg}"
        ((PASS++))
    else
        if [ "$status_pass" = false ]; then
            echo -e "  ${RED}✗ FAIL${NC} (Expected: $expected_status, Got: $http_code)${verify_msg}"
        else
            echo -e "  ${RED}✗ FAIL${NC} (Status: $http_code)${verify_msg}"
        fi
        ((FAIL++))
    fi
    
    # Show response body if present
    if [ -n "$body" ]; then
        echo "  Response: ${body:0:200}"
    fi
}

# Function to test with authentication (using purch_token cookie)
test_with_auth() {
    local method=$1
    local endpoint=$2
    local expected_status=$3
    local token=$4
    local data=$5
    local description=$6
    
    echo -e "\n${YELLOW}Testing:${NC} $description"
    echo "  ${method} ${BASE_URL}${endpoint}"
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -b "purch_token=$token" \
            -d "$data" \
            "${BASE_URL}${endpoint}")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -b "purch_token=$token" \
            "${BASE_URL}${endpoint}")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -eq "$expected_status" ]; then
        echo -e "  ${GREEN}✓ PASS${NC} (Status: $http_code)"
        ((PASS++))
        if [ -n "$body" ]; then
            echo "  Response: ${body:0:100}"
        fi
    else
        echo -e "  ${RED}✗ FAIL${NC} (Expected: $expected_status, Got: $http_code)"
        ((FAIL++))
    fi
}

# Function to register and login a user
register_and_login() {
    local first_name=$1
    local last_name=$2
    local username=$3
    local password=$4
    local income=$5
    local income_rate=$6
    
    echo -e "\n${YELLOW}Setting up test user...${NC}"
    
    # Build registration JSON
    local reg_json="{\"first_name\":\"$first_name\",\"last_name\":\"$last_name\",\"username\":\"$username\",\"password\":\"$password\""
    if [ -n "$income" ]; then
        reg_json="$reg_json,\"income\":$income"
    fi
    if [ -n "$income_rate" ]; then
        reg_json="$reg_json,\"income_rate\":\"$income_rate\""
    fi
    reg_json="$reg_json}"
    
    # Register user
    register_response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d "$reg_json" \
        "${BASE_URL}/user/register")
    
    register_code=$(echo "$register_response" | tail -n1)
    
    if [ "$register_code" -eq 201 ] || [ "$register_code" -eq 200 ]; then
        echo -e "  ${GREEN}✓${NC} User registered successfully"
    else
        echo -e "  ${YELLOW}⚠${NC} Registration returned $register_code (user may already exist)"
    fi
    
    # Login to get cookie
    login_response=$(curl -s -w "\n%{http_code}" \
        -H "Content-Type: application/json" \
        -b "$COOKIE_JAR" \
        -c "$COOKIE_JAR" \
        "${BASE_URL}/user/login?username=$username&password=$password")
    
    login_code=$(echo "$login_response" | tail -n1)
    login_body=$(echo "$login_response" | sed '$d')
    
    if [ "$login_code" -eq 200 ]; then
        echo -e "  ${GREEN}✓${NC} Login successful - purch_token cookie saved"
        echo "  Login response: ${login_body:0:100}"
        return 0
    else
        echo -e "  ${RED}✗${NC} Login failed with status $login_code"
        echo "  Response: $login_body"
        return 1
    fi
}

# Function to view current cookies in jar
show_cookies() {
    echo -e "\n${YELLOW}Current cookies in jar:${NC}"
    if [ -f "$COOKIE_JAR" ]; then
        cat "$COOKIE_JAR"
        echo ""
        echo -e "${YELLOW}purch_token cookie specifically:${NC}"
        grep 'purch_token' "$COOKIE_JAR" || echo "  ${RED}No purch_token found!${NC}"
    else
        echo "  ${RED}No cookie jar file found at $COOKIE_JAR${NC}"
    fi
}

echo "========================================="
echo "API Endpoint Testing Suite"
echo "Base URL: $BASE_URL"
echo "========================================="

# Register and login to get authenticated cookie
# Adjust credentials as needed
TEST_FIRST_NAME="foo"
TEST_LAST_NAME="bar"
TEST_USERNAME="testuser_$(date +%s)"
TEST_PASSWORD="testpass123"
TEST_INCOME="50000"
TEST_INCOME_RATE="yearly"

register_and_login "$TEST_FIRST_NAME" "$TEST_LAST_NAME" "$TEST_USERNAME" "$TEST_PASSWORD" "$TEST_INCOME" "$TEST_INCOME_RATE"

# Show the cookie that was saved
show_cookies

# Example tests - customize these for your API

# Unauthenticated endpoints (pass false as 6th parameter)
test_endpoint "GET" "/ping" 200 "" "Health check endpoint" false

# Authenticated endpoints with response verification examples
# The 7th parameter is a jq expression that should evaluate to true

# Verify specific user has correct structure
test_endpoint "GET" "/user/info" 200 "" \
    "Get specific user (authenticated)" \
    true \
    'has("ID") and has("Username") and has("FirstName") and has("LastName") and has("Password")'

# Verify we can get a link-token
test_endpoint "GET" "/user/link-token" 200 "" \
    "Get link token for current logged in user (authenticated)" \
    true \
    'has("LinkToken") and has("ExpiresAt")'

# Update user and verify response
test_endpoint "PUT" "/user/update" 200 \
    '{"first_name":"Jane","last_name":"Doe","income":75000, "income_rate":"weekly", "username":"othertestuser", "password":"testpass"}' \
    "Update user (authenticated)" \
    true \
    'has("message") and has("user")'

# Test deleting user returns works and returns message
test_endpoint "DELETE" "/user/delete" 200 "" \
    "Delete current user (authenticated)" \
    true \
    'has("message")'
    
# Test logging out
test_endpoint "GET" "/user/logout" 200 "" \
    "Logout and delete current cookie (authenticated)" \
    true \
    'has("message")'

# Test what happens without auth (should fail)
echo -e "\n${YELLOW}--- Testing without authentication ---${NC}"
rm -f "$COOKIE_JAR"  # Remove cookie
test_endpoint "GET" "/user/info" 401 "" "Get users without auth (should fail)" false

# Example with authentication using purch_token cookie
# The cookie is now automatically persisted from the login step above
# All test_endpoint calls will include the cookie automatically

# If you want to test without authentication, delete the cookie file first:
# rm -f "$COOKIE_JAR"

# Summary
echo -e "\n========================================="
echo -e "Test Results: ${GREEN}${PASS} passed${NC}, ${RED}${FAIL} failed${NC}"
echo "========================================="

exit $FAIL