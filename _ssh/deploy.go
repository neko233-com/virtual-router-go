package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	OS         string `yaml:"os"`
	Arch       string `yaml:"arch"`
	TargetDir  string `yaml:"target_dir"`
	BinaryName string `yaml:"binary_name"`
	BuildPath  string `yaml:"build_path"`
	StartCmd   string `yaml:"start_cmd"`
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	fmt.Printf("开始构建 %s (%s/%s)...\n", config.BinaryName, config.OS, config.Arch)
	localPath, err := buildBinary(config)
	if err != nil {
		log.Fatalf("构建失败: %v", err)
	}
	defer os.Remove(localPath)

	fmt.Printf("连接到服务器 %s:%d...\n", config.Host, config.Port)
	client, err := dialSSH(config)
	if err != nil {
		log.Fatalf("SSH连接失败: %v", err)
	}
	defer client.Close()

	fmt.Println("停止已有进程...")
	stopProcess(client, config.BinaryName)

	fmt.Printf("上传二进制文件到 %s...\n", config.TargetDir)
	err = uploadFile(client, localPath, config.TargetDir, config.BinaryName)
	if err != nil {
		log.Fatalf("上传失败: %v", err)
	}

	fmt.Println("正在启动服务...")
	err = runRemote(client, fmt.Sprintf("cd %s && chmod +x ./%s && %s", config.TargetDir, config.BinaryName, config.StartCmd))
	if err != nil {
		log.Fatalf("启动失败: %v", err)
	}

	fmt.Println("部署成功！")
}

func loadConfig() (*Config, error) {
	baseCfg := "_ssh/config_for_ssh_deploy.yaml"
	localCfg := "_ssh/config_for_ssh_deploy.local.yaml"

	cfg := &Config{
		Port: 22,
	}

	// Load base
	if err := readYaml(baseCfg, cfg); err != nil {
		return nil, err
	}

	// Load local
	if _, err := os.Stat(localCfg); err == nil {
		if err := readYaml(localCfg, cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func readYaml(path string, cfg *Config) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return yaml.NewDecoder(f).Decode(cfg)
}

func buildBinary(cfg *Config) (string, error) {
	tempName := cfg.BinaryName + "_tmp"
	if cfg.OS == "windows" {
		tempName += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", tempName, cfg.BuildPath)
	cmd.Env = append(os.Environ(),
		"GOOS="+cfg.OS,
		"GOARCH="+cfg.Arch,
		"CGO_ENABLED=0",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	absPath, _ := filepath.Abs(tempName)
	return absPath, nil
}

func dialSSH(cfg *Config) (*ssh.Client, error) {
	clientConfig := &ssh.ClientConfig{
		User: cfg.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(cfg.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	return ssh.Dial("tcp", addr, clientConfig)
}

func stopProcess(client *ssh.Client, name string) {
	// 尝试 kill 进程，不检查错误（可能没在跑）
	_ = runRemote(client, fmt.Sprintf("pkill -9 %s", name))
}

func uploadFile(client *ssh.Client, localPath, targetDir, name string) error {
	// 创建目录
	_ = runRemote(client, fmt.Sprintf("mkdir -p %s", targetDir))

	// 使用 cat 方式上传（简单无需 SFTP 库）
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	targetPath := filepath.Join(targetDir, name)
	// Linux path separator
	targetPath = strings.ReplaceAll(targetPath, "\\", "/")

	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	session.Stdin = f
	cmd := fmt.Sprintf("cat > %s", targetPath)
	return session.Run(cmd)
}

func runRemote(client *ssh.Client, cmd string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	return session.Run(cmd)
}
