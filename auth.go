package main

import (
    "crypto/rand"
    "encoding/json"
    "encoding/hex"
    "fmt"
    "github.com/dgrijalva/jwt-go"
    "io/ioutil"
    "net/http"
    "net/url"
    "os"
    "strings"
    "time"
)

var jsonPath string = "266235717_6h7w9t75_config.json"
var privateKeyPath string =  "private-decrypted.key"

type BoxToken struct {
    AccessToken  string   `json:"access_token"`
    ExpiresIn    int      `json:"expires_in"`
    RestrictedTo []string `json:"restricted_to"`
    TokenType    string   `json:"token_type"`
}

func auth() string {
    // Read JSON File
     jsonFile, err := os.Open(jsonPath)
     if err != nil {
         fmt.Println(err)
     }

     fmt.Println("Successfully Opened ...json")

     defer jsonFile.Close()

     byteValue, _ := ioutil.ReadAll(jsonFile)
     var config map[string]interface{}
     json.Unmarshal([]byte(byteValue), &config)
     setings := config["boxAppSettings"].(map[string]interface{})
     appAuth := setings["appAuth"].(map[string]interface{})
     fmt.Println("ClientID:", setings["clientID"])

    // Create JWT assertion
    token := jwt.NewWithClaims(jwt.SigningMethodRS512, jwt.MapClaims{
        "iss":          setings["clientID"],
        "sub":          config["enterpriseID"],
        "box_sub_type": "enterprise",
        "aud":          "https://api.box.com/oauth2/token",
        "jti":          GenerateJTI(64),
        "exp":          time.Now().Unix() + 60,
    })
    token.Header["kid"] = appAuth["publicKeyID"]

    privateKeyData, err := ioutil.ReadFile(privateKeyPath)
    if err != nil {
        panic(err)
    }

    key, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
    if err != nil {
        panic(err)
    }

    // Generate JTI request and store shortlived JTI token 
    tokenStr, err := token.SignedString(key)
    if err != nil {
        panic(err)
    }

    // Gnerate Token Request
    values := url.Values{}
    values.Add("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
    values.Add("client_id", setings["clientID"].(string))
    values.Add("client_secret", setings["clientSecret"].(string))
    values.Add("assertion", tokenStr)

    // Send Token Request
    req, err := http.NewRequest(http.MethodPost, "https://api.box.com/oauth2/token", strings.NewReader(values.Encode()))
    if err != nil {
        panic(err)
    }

    client := http.DefaultClient
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }

    defer resp.Body.Close()

    fmt.Println("OAuth Token Request Status:", resp.StatusCode)

    // Parse response and store BoxToken  
    responseBody, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        panic(err)
    }

    var boxToken BoxToken
    if err := json.Unmarshal(responseBody, &boxToken); err != nil {
        panic(err)
    }
    fmt.Println("OAuth Token:", boxToken.AccessToken)
    
    return boxToken.AccessToken
}

// Helper function to generate unique JTI
func GenerateJTI(length int) string {
    b := make([]byte, length)
    if _, err := rand.Read(b); err != nil {
        return ""
    }
    return hex.EncodeToString(b)
}