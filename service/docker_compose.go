package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/initia-labs/weave/common"
)

// Docker Compose methods for Rollytics
func (d *Docker) getComposeFilePath() (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userHome, common.RollyticsDirectory, "docker-compose.yml"), nil
}

func (d *Docker) getComposeProjectDir() (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userHome, common.RollyticsDirectory), nil
}

func (d *Docker) createDockerCompose(version string) error {
	composePath, err := d.getComposeFilePath()
	if err != nil {
		return fmt.Errorf("failed to get compose file path: %v", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(composePath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Read .env file to get configuration values
	userHome, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home: %v", err)
	}
	envPath := filepath.Join(userHome, common.RollyticsConfigPath)

	// Parse .env file to extract values
	envVars := make(map[string]string)
	if _, err := os.Stat(envPath); err == nil {
		envContent, err := os.ReadFile(envPath)
		if err == nil {
			lines := strings.Split(string(envContent), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					// Remove quotes if present
					value = strings.Trim(value, "\"")
					envVars[key] = value
				}
			}
		}
	}

	// Get image name
	imageName, err := d.getImageName(version)
	if err != nil {
		return fmt.Errorf("failed to get image name: %v", err)
	}

	// Generate docker-compose.yml
	const composeTemplate = `services:
  postgres:
    image: postgres:15-alpine
    container_name: rollytics-postgres
    env_file:
      - .env
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-postgres}
      POSTGRES_DB: ${POSTGRES_DB:-postgres}
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-postgres}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - rollytics-network

  rollytics-api:
    image: {{.ImageName}}
    container_name: rollytics-api
    command: ["api"]
    env_file:
      - .env
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      # Database configuration
      DB_DSN: postgres://${POSTGRES_USER:-postgres}:${POSTGRES_PASSWORD:-postgres}@postgres:5432/${POSTGRES_DB:-postgres}?sslmode=disable
      # Chain configuration - REQUIRED
      CHAIN_ID: ${CHAIN_ID}
      VM_TYPE: ${VM_TYPE}
      RPC_URL: ${RPC_URL}
      REST_URL: ${REST_URL}
      JSON_RPC_URL: ${JSON_RPC_URL}
      CORS_ENABLED: true
      CORS_ALLOW_ORIGINS: "*"
      CORS_ALLOW_METHODS: "GET,POST,PUT,DELETE,PATCH,OPTIONS,HEAD"
      CORS_ALLOW_HEADERS: "Origin, Content-Type, Accept, Authorization, X-Requested-With"
      CORS_ALLOW_CREDENTIALS: true
      CORS_MAX_AGE: 300
    ports:
      - "6767:8080"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    networks:
      - rollytics-network
    restart: always

  rollytics-indexer:
    image: {{.ImageName}}
    container_name: rollytics-indexer
    command: ["indexer"]
    env_file:
      - .env
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      # Database configuration
      DB_DSN: postgres://${POSTGRES_USER:-postgres}:${POSTGRES_PASSWORD:-postgres}@postgres:5432/${POSTGRES_DB:-postgres}?sslmode=disable
      # Chain configuration - REQUIRED
      CHAIN_ID: ${CHAIN_ID}
      VM_TYPE: ${VM_TYPE}
      RPC_URL: ${RPC_URL}
      REST_URL: ${REST_URL}
      JSON_RPC_URL: ${JSON_RPC_URL}
    extra_hosts:
      - "host.docker.internal:host-gateway"
    networks:
      - rollytics-network
    restart: unless-stopped

volumes:
  postgres_data:
    driver: local

networks:
  rollytics-network:
    driver: bridge
`

	tmpl, err := template.New("docker-compose").Parse(composeTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	file, err := os.Create(composePath)
	if err != nil {
		return fmt.Errorf("failed to create compose file: %v", err)
	}
	defer file.Close()

	data := struct {
		ImageName string
	}{
		ImageName: imageName,
	}

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return nil
}

func (d *Docker) startDockerCompose() error {
	projectDir, err := d.getComposeProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get compose project dir: %v", err)
	}

	composePath, err := d.getComposeFilePath()
	if err != nil {
		return fmt.Errorf("failed to get compose file path: %v", err)
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home: %v", err)
	}
	envPath := filepath.Join(userHome, common.RollyticsConfigPath)

	// Validate docker compose configuration before starting
	fmt.Println("Validating docker compose configuration...")
	validateCmd := exec.Command("docker", "compose", "-f", composePath, "--env-file", envPath, "config")
	validateCmd.Dir = projectDir
	validateCmd.Stderr = os.Stderr
	if err := validateCmd.Run(); err != nil {
		return fmt.Errorf("docker compose configuration is invalid: %v. Please check your docker-compose.yml and .env files", err)
	}

	// Start the services
	cmd := exec.Command("docker", "compose", "-f", composePath, "--env-file", envPath, "up", "-d")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start docker compose: %v", err)
	}

	return nil
}

func (d *Docker) stopDockerCompose() error {
	projectDir, err := d.getComposeProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get compose project dir: %v", err)
	}

	cmd := exec.Command("docker", "compose", "stop")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop docker compose: %v", err)
	}

	return nil
}

func (d *Docker) logDockerCompose(n int) error {
	projectDir, err := d.getComposeProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get compose project dir: %v", err)
	}

	cmd := exec.Command("docker", "compose", "logs", "-f", "--tail", fmt.Sprintf("%d", n))
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get docker compose logs: %v", err)
	}

	return nil
}

func (d *Docker) restartDockerCompose() error {
	projectDir, err := d.getComposeProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get compose project dir: %v", err)
	}

	cmd := exec.Command("docker", "compose", "restart")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart docker compose: %v", err)
	}

	return nil
}

func (d *Docker) removeDockerComposeVolumes() error {
	projectDir, err := d.getComposeProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get compose project dir: %v", err)
	}

	cmd := exec.Command("docker", "compose", "down", "-v")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove docker compose volumes: %v", err)
	}

	return nil
}
