//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

const baseURL = "http://localhost:8080/api/v1"

func main() {
	// Register admin user
	admin := map[string]string{
		"email":     "admin@gobank.io",
		"password":  "AdminPassword123!",
		"full_name": "System Admin",
		"phone":     "+12025550001",
	}
	registerAndPrint("Admin", admin)

	// Register test customer
	customer := map[string]string{
		"email":     "customer@gobank.io",
		"password":  "CustomerPass123!",
		"full_name": "Test Customer",
		"phone":     "+12025550002",
	}
	registerAndPrint("Customer", customer)

	fmt.Println("\nSeed complete. Use the returned user IDs to create accounts via the API.")
}

func registerAndPrint(name string, user map[string]string) {
	body, _ := json.Marshal(user)
	resp, err := http.Post(baseURL+"/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Fatalf("registering %s: %v", name, err)
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("[%s] Status: %d, ID: %v\n", name, resp.StatusCode, result["data"])
}
