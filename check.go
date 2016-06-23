package main

import (
	"encoding/json"
	"errors"
	"fmt"
	vault "github.com/hashicorp/vault/api"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	VaultURI string `json:"vault_uri"`
	AppID    string `json:"app_id,omitempty"`
	UserID   string // Should not be set from the passed config, is obtained from the environment
}

type Input struct {
	Source  Config            `json:"source"`
	Version Version           `json:"version"`
	Params  map[string]string `json:"params,omitempty"`
}

type Output struct {
	Version  Version    `json:"version"`
	Metadata []KeyValue `json:"metadata,omitempty"`
}

type Version struct {
	ExpirationTime string `json:"expires_at"`
}

type KeyValue struct {
	Key   string      `json:"name"`
	Value interface{} `json:"value"`
}

func main() {
	var err error

	input, command := ProcessInput()
	input.Source.UserID = os.Getenv("VAULT_USER_ID")

	switch command {
	case "check":
		version := Version{ExpirationTime: fmt.Sprintf("%d", time.Now().Unix())}
		err = json.NewEncoder(os.Stdout).Encode([1]Version{version})
		PanicCheck("Error writing response to stdout", err)
	case "in":
		if len(os.Args) < 2 {
			log.Panicf("usage: %s <dest directory>\n", command)
		}
		dataDir := os.Args[1]

		secrets, err := GetSecrets(input.Params, input.Source)
		PanicCheck("Could not retrieve secrets", err)

		// All checks out, generate output.
		outputFile, err := os.Create(fmt.Sprintf("%s/secrets.yaml", dataDir))
		defer outputFile.Close()
		PanicCheck("Failed to create output file: %s", err)

		// Output file is a yaml file, but json is also valid yaml
		err = json.NewEncoder(outputFile).Encode(secrets)
		PanicCheck("Writing response to file", err)

		output := Output{Version{ExpirationTime: "123455"}, nil}
		err = json.NewEncoder(os.Stdout).Encode(output)
		PanicCheck("writing response to stdout", err)
	default:
		log.Fatal("Only supports check & in. Use symlinked commands.")
	}

}

// Takes a map of prefix => path, retrieves the secrets at each path and merges them into a single
// map with each key prefixed with the prefix for that path.
func GetSecrets(mapping map[string]string, config Config) (secrets map[string]string, err error) {
	vault := VaultSession(config)
	secrets = make(map[string]string)
	for prefix, path := range mapping {
		vaultSecret, err := vault.Logical().Read(path)

		// Error handling
		if err != nil {
			return secrets, err
		}
		if vaultSecret == nil {
			return secrets, errors.New(fmt.Sprintf("404 Secret @ `%s` not found!", path))
		}
		for warning := range vaultSecret.Warnings {
			log.Println(warning)
		}

		// Build prefixed secrets map
		for k, v := range vaultSecret.Data {
			key := fmt.Sprintf("%s-%s", prefix, k)
			secrets[key] = v.(string)
		}
	}
	return
}

// Setup a new Vault session & authenticate it with App ID
func VaultSession(config Config) *vault.Client {
	// Check prerequisites
	if config.AppID == "" {
		log.Panicln("UserID not set")
	}
	if config.UserID == "" {
		log.Panicln("AppID not set")
	}
	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = config.VaultURI
	client, err := vault.NewClient(vaultConfig)
	PanicCheck("Could not setup Vault client", err)

	// Authenticate
	loginData := map[string]interface{}{"app_id": config.AppID, "user_id": config.UserID}
	secret, err := client.Logical().Write("auth/app-id/login", loginData)
	PanicCheck("Could not authenticate with vault", err)
	client.SetToken(secret.Auth.ClientToken)

	return client
}

func ProcessInput() (Input, string) {
	stdin, err := ioutil.ReadAll(os.Stdin)
	PanicCheck("Reading request from stdin", err)

	var input Input
	err = json.Unmarshal(stdin, &input)
	PanicCheck("Error parsing input", err)
	command := filepath.Base(os.Args[0])
	return input, command
}

func PanicCheck(msg string, err error) {
	if err != nil {
		log.Panicf("%s:\n%s", msg, err)
	}
}
