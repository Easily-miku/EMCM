package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	API_BASE      = "https://download.fastmirror.net/api/v3"
	CACHE_DIR     = ".emcm"
	DICT_FILE     = "logs.dict"
	CONFIG_FILE   = "emcm.config"
	SERVERS_FILE  = "servers.json"
	MAX_API_CALLS = 200
	MAX_SERVERS   = 10
)

var (
	serverList     []ServerInfo
	translationMap = make(map[string]string)
	config         Config
	runningServers = make(map[string]*exec.Cmd)
	serverMutex    sync.Mutex
)

type ServerInfo struct {
	Name      string   `json:"name"`
	Tag       string   `json:"tag"`
	Recommend bool     `json:"recommend"`
	Versions  []string `json:"mc_versions"`
}

type BuildInfo struct {
	Builds []struct {
		Name       string `json:"name"`
		MCVersion  string `json:"mc_version"`
		Core       string `json:"core_version"`
		UpdateTime string `json:"update_time"`
		SHA1       string `json:"sha1"`
	} `json:"builds"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
	Count  int `json:"count"`
}

type CoreMetadata struct {
	Name        string `json:"name"`
	MCVersion   string `json:"mc_version"`
	CoreVersion string `json:"core_version"`
	UpdateTime  string `json:"update_time"`
	SHA1        string `json:"sha1"`
	Filename    string `json:"filename"`
	DownloadURL string `json:"download_url"`
}

type ServerInstance struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ServerType  string `json:"server_type"`
	MCVersion   string `json:"mc_version"`
	CoreVersion string `json:"core_version"`
	Path        string `json:"path"`
	JavaPath    string `json:"java_path"`
	Memory      int    `json:"memory"` // MB
	JVMArgs     string `json:"jvm_args"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Config struct {
	JavaPath       string                     `json:"java_path"`
	JavaVersions   map[string]string          `json:"java_versions"`
	DefaultMemory  int                        `json:"default_memory"`
	ServerInstalls map[string]*ServerInstance `json:"server_installs"`
	APICalls       int                        `json:"api_calls"`
	LastAPICall    time.Time                  `json:"last_api_call"`
}

func main() {
	initApp()
	displayBanner()
	loadConfig()
	loadTranslationDict()

	if len(os.Args) > 1 {
		handleCLI()
	} else {
		showMainMenu()
	}
}

func initApp() {
	os.Mkdir(CACHE_DIR, 0755)
	os.Mkdir(filepath.Join(CACHE_DIR, "servers"), 0755)
	os.Mkdir(filepath.Join(CACHE_DIR, "cache"), 0755)
	os.Mkdir(filepath.Join(CACHE_DIR, "java"), 0755)
}

func displayBanner() {
	colorReset := "\033[0m"
	colorCyan := "\033[36m"
	colorYellow := "\033[33m"

	fmt.Println(colorCyan)
	fmt.Println("███████╗███╗   ███╗ ██████╗███╗   ███╗")
	fmt.Println("██╔════╝████╗ ████║██╔════╝████╗ ████║")
	fmt.Println("█████╗  ██╔████╔██║██║     ██╔████╔██║")
	fmt.Println("██╔══╝  ██║╚██╔╝██║██║     ██║╚██╔╝██║")
	fmt.Println("███████╗██║ ╚═╝ ██║╚██████╗██║ ╚═╝ ██║")
	fmt.Println("╚══════╝╚═╝     ╚═╝ ╚═════╝╚═╝     ╚═╝")
	fmt.Println("                                      ")
	fmt.Println("Easily Minecraft Manager v2.1")
	fmt.Println(colorYellow + "Author: Easily-Miku")
	fmt.Println("GitHub: https://github.com/Easily-miku")
	fmt.Println(colorCyan + "--------------------------------------")
	fmt.Println(colorReset)
}

func loadConfig() {
	configPath := filepath.Join(CACHE_DIR, CONFIG_FILE)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config = Config{
			JavaPath:       detectJava(),
			DefaultMemory:  2048,
			JavaVersions:   make(map[string]string),
			ServerInstalls: make(map[string]*ServerInstance),
			APICalls:       0,
			LastAPICall:    time.Now(),
		}
		saveConfig()
		return
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatalf("解析配置失败: %v", err)
	}

	if time.Since(config.LastAPICall) > time.Hour {
		config.APICalls = 0
		config.LastAPICall = time.Now()
		saveConfig()
	}

	if config.JavaVersions == nil {
		config.JavaVersions = make(map[string]string)
	}
}

func saveConfig() {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Printf("保存配置失败: %v", err)
		return
	}

	configPath := filepath.Join(CACHE_DIR, CONFIG_FILE)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		log.Printf("写入配置文件失败: %v", err)
	}
}

func loadTranslationDict() {
	dictPath := filepath.Join(CACHE_DIR, DICT_FILE)
	if _, err := os.Stat(dictPath); os.IsNotExist(err) {
		defaultDict := []byte(`Player [a-zA-Z0-9_]+ joined#玩家 $0 加入游戏
Done \(\d+\.\d+s\)!#启动完成 (耗时 $0 秒)
Stopping server#正在停止服务器
Preparing spawn area: (\d+)%#生成出生点区域: $1%`)
		if err := os.WriteFile(dictPath, defaultDict, 0644); err != nil {
			log.Printf("创建默认字典失败: %v", err)
		}
	}

	file, err := os.Open(dictPath)
	if err != nil {
		log.Printf("打开字典文件失败: %v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "#", 2)
		if len(parts) == 2 {
			translationMap[parts[0]] = parts[1]
		}
	}
}

func translateLog(line string) string {
	for pattern, translation := range translationMap {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(line); matches != nil {
			result := translation
			for i, match := range matches {
				placeholder := fmt.Sprintf("$%d", i)
				result = strings.ReplaceAll(result, placeholder, match)
			}
			return result
		}
	}
	return line
}

func detectJava() string {
	if runtime.GOOS == "windows" {
		if path, err := exec.LookPath("javaw.exe"); err == nil {
			return path
		}
	} else {
		if path, err := exec.LookPath("java"); err == nil {
			return path
		}
	}
	return ""
}

func recommendJavaVersion(mcVersion string) string {
	versionParts := strings.Split(mcVersion, ".")
	if len(versionParts) < 2 {
		return config.JavaPath
	}

	major, _ := strconv.Atoi(versionParts[0])
	minor, _ := strconv.Atoi(versionParts[1])

	if major >= 1 && minor >= 17 {
		if path, ok := config.JavaVersions["17"]; ok {
			return path
		}
		return "java17"
	} else if major >= 1 && minor >= 12 {
		if path, ok := config.JavaVersions["11"]; ok {
			return path
		}
		return "java11"
	} else {
		if path, ok := config.JavaVersions["8"]; ok {
			return path
		}
		return "java8"
	}
}

func apiGet(path string, target interface{}) error {
	if config.APICalls >= MAX_API_CALLS {
		return errors.New("API调用次数已达上限，请稍后再试")
	}

	fullURL := API_BASE + path
	resp, err := http.Get(fullURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API请求失败: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var apiResponse struct {
		Data    json.RawMessage `json:"data"`
		Code    string          `json:"code"`
		Success bool            `json:"success"`
		Message string          `json:"message"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return err
	}

	if !apiResponse.Success {
		return fmt.Errorf("API错误: %s", apiResponse.Message)
	}

	config.APICalls++
	saveConfig()

	return json.Unmarshal(apiResponse.Data, target)
}

func getServerList() ([]ServerInfo, error) {
	if len(serverList) > 0 {
		return serverList, nil
	}

	cachePath := filepath.Join(CACHE_DIR, "cache", "servers.json")
	if _, err := os.Stat(cachePath); err == nil {
		data, err := os.ReadFile(cachePath)
		if err == nil {
			if json.Unmarshal(data, &serverList) == nil {
				return serverList, nil
			}
		}
	}

	if err := apiGet("", &serverList); err != nil {
		return nil, err
	}

	data, _ := json.Marshal(serverList)
	os.WriteFile(cachePath, data, 0644)

	return serverList, nil
}

func getServerInfo(name string) (*ServerInfo, error) {
	servers, err := getServerList()
	if err != nil {
		return nil, err
	}

	for _, s := range servers {
		if strings.EqualFold(s.Name, name) {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("未找到服务端: %s", name)
}

func getBuilds(name, mcVersion string) (*BuildInfo, error) {
	path := fmt.Sprintf("/%s/%s", url.PathEscape(name), url.PathEscape(mcVersion))
	var builds BuildInfo
	if err := apiGet(path, &builds); err != nil {
		return nil, err
	}
	return &builds, nil
}

func getCoreMetadata(name, mcVersion, coreVersion string) (*CoreMetadata, error) {
	path := fmt.Sprintf("/%s/%s/%s", url.PathEscape(name), url.PathEscape(mcVersion), url.PathEscape(coreVersion))
	var metadata CoreMetadata
	if err := apiGet(path, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

func downloadServer(name, mcVersion, coreVersion string) (string, error) {
	metadata, err := getCoreMetadata(name, mcVersion, coreVersion)
	if err != nil {
		return "", err
	}

	serverDir := filepath.Join(CACHE_DIR, "servers", fmt.Sprintf("%s-%s", name, mcVersion))
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		return "", err
	}

	filePath := filepath.Join(serverDir, metadata.Filename)
	if _, err := os.Stat(filePath); err == nil {
		return filePath, nil
	}

	resp, err := http.Get(metadata.DownloadURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败: %s", resp.Status)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

func startServer(serverID string) {
	server, ok := config.ServerInstalls[serverID]
	if !ok {
		fmt.Printf("找不到服务器实例: %s\n", serverID)
		return
	}

	javaPath := server.JavaPath
	if javaPath == "" {
		javaPath = config.JavaPath
	}
	if javaPath == "" {
		fmt.Println("未找到Java环境，请先配置Java路径")
		return
	}

	memory := fmt.Sprintf("%dM", server.Memory)
	args := []string{
		"-Xms" + memory,
		"-Xmx" + memory,
		"-XX:+UseG1GC",
		"-jar",
		server.Path,
		"nogui",
	}

	if server.JVMArgs != "" {
		extraArgs := strings.Split(server.JVMArgs, " ")
		args = append(args, extraArgs...)
	}

	cmd := exec.Command(javaPath, args...)
	cmd.Dir = filepath.Dir(server.Path)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// 保存运行中的服务器
	serverMutex.Lock()
	runningServers[serverID] = cmd
	serverMutex.Unlock()

	colorGreen := "\033[32m"
	colorReset := "\033[0m"
	fmt.Printf("%s服务器 [%s] 启动中... (输入 'stop' 停止服务器)%s\n", colorGreen, server.Name, colorReset)

	// 输出处理
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("[%s] %s\n", server.Name, translateLog(line))
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("[%s] %s\n", server.Name, translateLog(line))
		}
	}()

	// 用户输入处理
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.EqualFold(text, "stop") {
			fmt.Fprintln(stdin, "stop")
			break
		}
		fmt.Fprintln(stdin, text)
	}

	cmd.Wait()

	// 清理运行中的服务器
	serverMutex.Lock()
	delete(runningServers, serverID)
	serverMutex.Unlock()

	fmt.Printf("%s服务器 [%s] 已停止%s\n", colorGreen, server.Name, colorReset)
}

func stopServer(serverID string) {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if cmd, ok := runningServers[serverID]; ok {
		cmd.Process.Signal(os.Interrupt)
		fmt.Printf("已发送停止信号到服务器: %s\n", serverID)
	} else {
		fmt.Printf("未找到运行中的服务器: %s\n", serverID)
	}
}

func handleCLI() {
	if len(os.Args) < 2 {
		showMainMenu()
		return
	}

	switch os.Args[1] {
	case "list":
		servers, err := getServerList()
		if err != nil {
			fmt.Println("错误:", err)
			return
		}
		fmt.Println("\n可用服务端:")
		for _, s := range servers {
			rec := ""
			if s.Recommend {
				rec = " (推荐)"
			}
			fmt.Printf("- %s%s [%s]\n", s.Name, rec, s.Tag)
		}

	case "versions":
		if len(os.Args) < 3 {
			fmt.Println("用法: emcm versions <服务端名称>")
			return
		}
		server, err := getServerInfo(os.Args[2])
		if err != nil {
			fmt.Println("错误:", err)
			return
		}
		fmt.Printf("\n%s 支持的MC版本:\n", server.Name)
		for _, v := range server.Versions {
			fmt.Println("-", v)
		}

	case "download":
		if len(os.Args) < 4 {
			fmt.Println("用法: emcm download <服务端名称> <MC版本> [核心版本]")
			return
		}
		name := os.Args[2]
		mcVersion := os.Args[3]
		coreVersion := ""
		if len(os.Args) > 4 {
			coreVersion = os.Args[4]
		}

		if coreVersion == "" {
			builds, err := getBuilds(name, mcVersion)
			if err != nil {
				fmt.Println("错误:", err)
				return
			}
			if len(builds.Builds) == 0 {
				fmt.Println("未找到可用构建")
				return
			}
			coreVersion = builds.Builds[0].Core
			fmt.Printf("使用最新版本: %s\n", coreVersion)
		}

		path, err := downloadServer(name, mcVersion, coreVersion)
		if err != nil {
			fmt.Println("下载失败:", err)
			return
		}
		fmt.Printf("下载完成! 文件保存至: %s\n", path)

		// 创建服务器实例
		serverID := fmt.Sprintf("server-%d", len(config.ServerInstalls)+1)
		server := &ServerInstance{
			ID:          serverID,
			Name:        fmt.Sprintf("%s-%s", name, mcVersion),
			ServerType:  name,
			MCVersion:   mcVersion,
			CoreVersion: coreVersion,
			Path:        path,
			JavaPath:    recommendJavaVersion(mcVersion),
			Memory:      config.DefaultMemory,
			CreatedAt:   time.Now().Format(time.RFC3339),
			UpdatedAt:   time.Now().Format(time.RFC3339),
		}
		config.ServerInstalls[serverID] = server
		saveConfig()
		fmt.Printf("已创建服务器实例: %s\n", serverID)

	case "start":
		if len(os.Args) < 3 {
			fmt.Println("用法: emcm start <服务器ID>")
			return
		}
		startServer(os.Args[2])

	case "stop":
		if len(os.Args) < 3 {
			fmt.Println("用法: emcm stop <服务器ID>")
			return
		}
		stopServer(os.Args[2])

	case "java":
		if len(os.Args) < 3 {
			fmt.Println("当前Java路径:", config.JavaPath)
			fmt.Println("已配置Java版本:")
			for ver, path := range config.JavaVersions {
				fmt.Printf("- Java %s: %s\n", ver, path)
			}
			return
		}
		switch os.Args[2] {
		case "set":
			if len(os.Args) < 4 {
				fmt.Println("用法: emcm java set <java路径>")
				return
			}
			config.JavaPath = os.Args[3]
			saveConfig()
			fmt.Println("Java路径已更新")
		case "detect":
			path := detectJava()
			if path == "" {
				fmt.Println("未检测到Java环境")
			} else {
				config.JavaPath = path
				saveConfig()
				fmt.Println("检测到Java:", path)
			}
		case "add":
			if len(os.Args) < 5 {
				fmt.Println("用法: emcm java add <版本> <路径>")
				return
			}
			version := os.Args[3]
			path := os.Args[4]
			config.JavaVersions[version] = path
			saveConfig()
			fmt.Printf("已添加Java %s: %s\n", version, path)
		}

	case "memory":
		if len(os.Args) < 3 {
			fmt.Printf("当前默认内存: %dMB\n", config.DefaultMemory)
			return
		}
		mem, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println("请输入有效的内存大小(MB)")
			return
		}
		config.DefaultMemory = mem
		saveConfig()
		fmt.Printf("默认内存已设置为 %dMB\n", mem)

	case "servers":
		fmt.Println("\n已安装的服务器:")
		for id, server := range config.ServerInstalls {
			fmt.Printf("- ID: %s\n  名称: %s\n  类型: %s %s\n  路径: %s\n",
				id, server.Name, server.ServerType, server.MCVersion, server.Path)
		}

	default:
		fmt.Println("未知命令:", os.Args[1])
		fmt.Println("可用命令: list, versions, download, start, stop, java, memory, servers")
	}
}

func showMainMenu() {
	for {
		clearScreen()
		displayBanner()

		if len(config.ServerInstalls) == 0 {
			fmt.Println("\033[33m警告: 尚未创建任何服务器实例\033[0m")
			fmt.Println("请先创建服务器实例")
			time.Sleep(2 * time.Second)
			createServerInstance()
			continue
		}

		fmt.Println("\n\033[1;36mEMCM 主菜单\033[0m")
		fmt.Println("----------------------------------------")
		fmt.Println("1. 启动服务器")
		fmt.Println("2. 停止服务器")
		fmt.Println("3. 管理服务器实例")
		fmt.Println("4. 下载服务端核心")
		fmt.Println("5. Java环境管理")
		fmt.Println("6. 内存设置")
		fmt.Println("7. 编辑日志翻译字典")
		fmt.Println("8. 退出")
		fmt.Println("----------------------------------------")
		fmt.Print("请选择操作: ")

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			startServerMenu()
		case 2:
			stopServerMenu()
		case 3:
			manageServersMenu()
		case 4:
			downloadServerMenu()
		case 5:
			javaManagementMenu()
		case 6:
			memorySettingsMenu()
		case 7:
			editTranslationDict()
		case 8:
			fmt.Println("感谢使用 EMCM!")
			os.Exit(0)
		default:
			fmt.Println("无效选择")
			time.Sleep(1 * time.Second)
		}
	}
}

func startServerMenu() {
	clearScreen()
	fmt.Println("\n\033[1;36m启动服务器\033[0m")
	fmt.Println("----------------------------------------")

	if len(config.ServerInstalls) == 0 {
		fmt.Println("没有可用的服务器实例")
		time.Sleep(2 * time.Second)
		return
	}

	// 显示服务器列表
	serverIDs := make([]string, 0, len(config.ServerInstalls))
	i := 1
	fmt.Println("选择要启动的服务器:")
	for id, server := range config.ServerInstalls {
		fmt.Printf("%d. %s (%s %s)\n", i, server.Name, server.ServerType, server.MCVersion)
		serverIDs = append(serverIDs, id)
		i++
	}
	fmt.Println("0. 返回")
	fmt.Println("----------------------------------------")
	fmt.Print("请选择: ")

	var choice int
	fmt.Scanln(&choice)

	if choice == 0 {
		return
	}

	if choice < 1 || choice > len(serverIDs) {
		fmt.Println("无效选择")
		time.Sleep(1 * time.Second)
		return
	}

	serverID := serverIDs[choice-1]
	server := config.ServerInstalls[serverID]
	fmt.Printf("启动服务器: %s\n", server.Name)

	// 检查Java环境
	if server.JavaPath == "" {
		recommended := recommendJavaVersion(server.MCVersion)
		if strings.HasPrefix(recommended, "java") {
			fmt.Printf("\033[33m警告: 推荐使用Java %s，但未配置。使用默认Java路径: %s\033[0m\n",
				strings.TrimPrefix(recommended, "java"), config.JavaPath)
		} else {
			server.JavaPath = recommended
		}
	}

	startServer(serverID)
}

func stopServerMenu() {
	clearScreen()
	fmt.Println("\n\033[1;36m停止服务器\033[0m")
	fmt.Println("----------------------------------------")

	if len(runningServers) == 0 {
		fmt.Println("没有运行中的服务器")
		time.Sleep(2 * time.Second)
		return
	}

	// 显示运行中的服务器
	serverIDs := make([]string, 0, len(runningServers))
	i := 1
	fmt.Println("运行中的服务器:")
	for id := range runningServers {
		if server, ok := config.ServerInstalls[id]; ok {
			fmt.Printf("%d. %s\n", i, server.Name)
			serverIDs = append(serverIDs, id)
			i++
		}
	}
	fmt.Println("0. 返回")
	fmt.Println("----------------------------------------")
	fmt.Print("请选择: ")

	var choice int
	fmt.Scanln(&choice)

	if choice == 0 {
		return
	}

	if choice < 1 || choice > len(serverIDs) {
		fmt.Println("无效选择")
		time.Sleep(1 * time.Second)
		return
	}

	serverID := serverIDs[choice-1]
	stopServer(serverID)
}

func manageServersMenu() {
	for {
		clearScreen()
		fmt.Println("\n\033[1;36m服务器实例管理\033[0m")
		fmt.Println("----------------------------------------")

		if len(config.ServerInstalls) == 0 {
			fmt.Println("没有可用的服务器实例")
			fmt.Println("----------------------------------------")
			fmt.Println("1. 创建新实例")
			fmt.Println("0. 返回")
			fmt.Println("----------------------------------------")
			fmt.Print("请选择: ")

			var choice int
			fmt.Scanln(&choice)

			if choice == 0 {
				return
			} else if choice == 1 {
				createServerInstance()
			}
			continue
		}

		// 显示服务器列表
		serverIDs := make([]string, 0, len(config.ServerInstalls))
		i := 1
		fmt.Println("服务器实例:")
		for id, server := range config.ServerInstalls {
			fmt.Printf("%d. %s (%s %s)\n", i, server.Name, server.ServerType, server.MCVersion)
			serverIDs = append(serverIDs, id)
			i++
		}
		fmt.Println("----------------------------------------")
		fmt.Println("1. 创建新实例")
		fmt.Println("2. 重命名实例")
		fmt.Println("3. 配置Java环境")
		fmt.Println("4. 配置启动参数")
		fmt.Println("5. 删除实例")
		fmt.Println("0. 返回")
		fmt.Println("----------------------------------------")
		fmt.Print("请选择操作: ")

		var action int
		fmt.Scanln(&action)

		if action == 0 {
			return
		}

		if action == 1 {
			createServerInstance()
			continue
		}

		if action >= 2 && action <= 5 {
			fmt.Print("请选择服务器实例: ")
			var serverChoice int
			fmt.Scanln(&serverChoice)

			if serverChoice < 1 || serverChoice > len(serverIDs) {
				fmt.Println("无效选择")
				time.Sleep(1 * time.Second)
				continue
			}

			serverID := serverIDs[serverChoice-1]
			server := config.ServerInstalls[serverID]

			switch action {
			case 2: // 重命名
				fmt.Printf("当前名称: %s\n", server.Name)
				fmt.Print("输入新名称: ")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				newName := scanner.Text()
				if newName != "" {
					server.Name = newName
					server.UpdatedAt = time.Now().Format(time.RFC3339)
					saveConfig()
					fmt.Println("名称已更新")
				}
			case 3: // 配置Java
				fmt.Printf("当前Java路径: %s\n", server.JavaPath)
				fmt.Println("可用Java版本:")
				for ver, path := range config.JavaVersions {
					fmt.Printf("- %s: %s\n", ver, path)
				}
				fmt.Print("输入Java版本或完整路径: ")
				var javaInput string
				fmt.Scanln(&javaInput)

				if path, ok := config.JavaVersions[javaInput]; ok {
					server.JavaPath = path
				} else {
					server.JavaPath = javaInput
				}
				server.UpdatedAt = time.Now().Format(time.RFC3339)
				saveConfig()
				fmt.Println("Java配置已更新")
			case 4: // 配置启动参数
				fmt.Printf("当前JVM参数: %s\n", server.JVMArgs)
				fmt.Print("输入新的JVM参数: ")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				newArgs := scanner.Text()
				server.JVMArgs = newArgs
				server.UpdatedAt = time.Now().Format(time.RFC3339)
				saveConfig()
				fmt.Println("启动参数已更新")
			case 5: // 删除
				fmt.Printf("确定要删除服务器实例 '%s' 吗? (y/n): ", server.Name)
				var confirm string
				fmt.Scanln(&confirm)
				if strings.ToLower(confirm) == "y" {
					delete(config.ServerInstalls, serverID)
					saveConfig()
					fmt.Println("实例已删除")
				}
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func createServerInstance() {
	clearScreen()
	fmt.Println("\n\033[1;36m创建服务器实例\033[0m")
	fmt.Println("----------------------------------------")

	// 1. 命名服务器实例
	fmt.Print("请输入服务器名称: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	serverName := scanner.Text()

	if serverName == "" {
		serverName = "未命名服务器"
	}

	// 2. 选择创建方式
	fmt.Println("\n请选择创建方式:")
	fmt.Println("1. 下载新服务端")
	fmt.Println("2. 使用现有服务端")
	fmt.Print("选择: ")

	var choice int
	fmt.Scanln(&choice)

	var serverPath, serverType, mcVersion, coreVersion string

	switch choice {
	case 1:
		path, st, mv, cv, err := downloadServerMenu()
		if err != nil {
			fmt.Printf("下载失败: %v\n", err)
			return
		}
		serverPath = path
		serverType = st
		mcVersion = mv
		coreVersion = cv
	case 2:
		fmt.Print("请输入服务端路径: ")
		fmt.Scanln(&serverPath)

		// 尝试解析服务端类型和版本
		fileName := filepath.Base(serverPath)
		if strings.Contains(fileName, "paper") {
			serverType = "Paper"
		} else if strings.Contains(fileName, "forge") {
			serverType = "Forge"
		} else if strings.Contains(fileName, "fabric") {
			serverType = "Fabric"
		} else {
			serverType = "Unknown"
		}

		// 从文件名中提取版本号
		re := regexp.MustCompile(`(\d+\.\d+(\.\d+)?)`)
		if matches := re.FindStringSubmatch(fileName); len(matches) > 0 {
			mcVersion = matches[0]
		} else {
			mcVersion = "Unknown"
		}
	default:
		fmt.Println("无效选择")
		return
	}

	// 3. 配置Java环境
	javaPath := config.JavaPath
	if mcVersion != "" && mcVersion != "Unknown" {
		recommended := recommendJavaVersion(mcVersion)
		if !strings.HasPrefix(recommended, "java") {
			javaPath = recommended
		}
	}

	// 创建服务器实例
	serverID := fmt.Sprintf("server-%d", len(config.ServerInstalls)+1)
	server := &ServerInstance{
		ID:          serverID,
		Name:        serverName,
		ServerType:  serverType,
		MCVersion:   mcVersion,
		CoreVersion: coreVersion,
		Path:        serverPath,
		JavaPath:    javaPath,
		Memory:      config.DefaultMemory,
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}

	config.ServerInstalls[serverID] = server
	saveConfig()

	fmt.Printf("\n\033[32m服务器实例创建成功!\033[0m\n")
	fmt.Printf("ID: %s\n", serverID)
	fmt.Printf("名称: %s\n", serverName)
	fmt.Printf("类型: %s\n", serverType)
	if mcVersion != "" {
		fmt.Printf("MC版本: %s\n", mcVersion)
	}
	fmt.Printf("路径: %s\n", serverPath)
	fmt.Printf("Java路径: %s\n", javaPath)
	fmt.Printf("内存: %dMB\n", config.DefaultMemory)

	fmt.Println("\n按回车键返回...")
	fmt.Scanln()
}

func downloadServerMenu() (string, string, string, string, error) {
	clearScreen()
	fmt.Println("\n\033[1;36m下载服务端\033[0m")
	fmt.Println("----------------------------------------")

	servers, err := getServerList()
	if err != nil {
		fmt.Println("获取服务端列表失败:", err)
		time.Sleep(2 * time.Second)
		return "", "", "", "", err
	}

	// 显示服务端列表
	fmt.Println("选择服务端:")
	for i, s := range servers {
		fmt.Printf("%d. %s (%s)\n", i+1, s.Name, s.Tag)
	}
	fmt.Println("0. 返回")
	fmt.Println("----------------------------------------")
	fmt.Print("请选择: ")

	var choice int
	fmt.Scanln(&choice)

	if choice == 0 {
		return "", "", "", "", errors.New("操作取消")
	}

	if choice < 1 || choice > len(servers) {
		return "", "", "", "", errors.New("无效选择")
	}

	selectedServer := servers[choice-1].Name
	server, err := getServerInfo(selectedServer)
	if err != nil {
		return "", "", "", "", err
	}

	// 选择版本
	clearScreen()
	fmt.Printf("\n\033[1;36m选择 %s 版本\033[0m\n", selectedServer)
	fmt.Println("----------------------------------------")
	fmt.Println("选择MC版本:")
	for i, version := range server.Versions {
		fmt.Printf("%d. %s\n", i+1, version)
	}
	fmt.Println("0. 返回")
	fmt.Println("----------------------------------------")
	fmt.Print("请选择: ")

	fmt.Scanln(&choice)

	if choice == 0 {
		return "", "", "", "", errors.New("操作取消")
	}

	if choice < 1 || choice > len(server.Versions) {
		return "", "", "", "", errors.New("无效选择")
	}

	selectedVersion := server.Versions[choice-1]

	// 获取构建版本
	builds, err := getBuilds(selectedServer, selectedVersion)
	if err != nil {
		return "", "", "", "", err
	}

	if len(builds.Builds) == 0 {
		return "", "", "", "", errors.New("未找到可用构建")
	}

	// 选择构建版本
	clearScreen()
	fmt.Printf("\n\033[1;36m选择 %s %s 构建版本\033[0m\n", selectedServer, selectedVersion)
	fmt.Println("----------------------------------------")
	fmt.Println("选择构建版本:")
	for i, build := range builds.Builds {
		fmt.Printf("%d. %s (更新时间: %s)\n", i+1, build.Core, build.UpdateTime)
	}
	fmt.Println("0. 返回")
	fmt.Println("----------------------------------------")
	fmt.Print("请选择: ")

	fmt.Scanln(&choice)

	if choice == 0 {
		return "", "", "", "", errors.New("操作取消")
	}

	if choice < 1 || choice > len(builds.Builds) {
		return "", "", "", "", errors.New("无效选择")
	}

	selectedBuild := builds.Builds[choice-1].Core

	// 下载服务端
	fmt.Printf("\n正在下载 %s %s (%s)...\n", selectedServer, selectedVersion, selectedBuild)
	path, err := downloadServer(selectedServer, selectedVersion, selectedBuild)
	if err != nil {
		return "", "", "", "", err
	}

	fmt.Printf("\033[32m下载完成! 文件保存至: %s\033[0m\n", path)
	return path, selectedServer, selectedVersion, selectedBuild, nil
}

func javaManagementMenu() {
	for {
		clearScreen()
		fmt.Println("\n\033[1;36mJava 环境管理\033[0m")
		fmt.Println("----------------------------------------")
		fmt.Printf("当前默认Java路径: %s\n", config.JavaPath)
		fmt.Println("已配置Java版本:")
		for ver, path := range config.JavaVersions {
			fmt.Printf("- Java %s: %s\n", ver, path)
		}
		fmt.Println("----------------------------------------")
		fmt.Println("1. 自动检测Java")
		fmt.Println("2. 设置默认Java路径")
		fmt.Println("3. 添加Java版本")
		fmt.Println("4. 删除Java版本")
		fmt.Println("0. 返回主菜单")
		fmt.Println("----------------------------------------")
		fmt.Print("请选择操作: ")

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 0:
			return
		case 1:
			path := detectJava()
			if path == "" {
				fmt.Println("未检测到Java环境")
			} else {
				config.JavaPath = path
				saveConfig()
				fmt.Printf("检测到Java: %s\n", path)
			}
			time.Sleep(2 * time.Second)
		case 2:
			fmt.Print("请输入Java完整路径: ")
			var path string
			fmt.Scanln(&path)
			if _, err := os.Stat(path); err == nil {
				config.JavaPath = path
				saveConfig()
				fmt.Println("默认Java路径已更新")
			} else {
				fmt.Println("路径无效或文件不存在")
			}
			time.Sleep(2 * time.Second)
		case 3:
			fmt.Print("请输入Java版本(如8,11,17): ")
			var version string
			fmt.Scanln(&version)
			fmt.Print("请输入Java完整路径: ")
			var path string
			fmt.Scanln(&path)
			if _, err := os.Stat(path); err == nil {
				config.JavaVersions[version] = path
				saveConfig()
				fmt.Printf("已添加Java %s: %s\n", version, path)
			} else {
				fmt.Println("路径无效或文件不存在")
			}
			time.Sleep(2 * time.Second)
		case 4:
			fmt.Print("请输入要删除的Java版本: ")
			var version string
			fmt.Scanln(&version)
			if _, ok := config.JavaVersions[version]; ok {
				delete(config.JavaVersions, version)
				saveConfig()
				fmt.Printf("已删除Java %s\n", version)
			} else {
				fmt.Println("未找到该版本")
			}
			time.Sleep(2 * time.Second)
		default:
			fmt.Println("无效选择")
			time.Sleep(1 * time.Second)
		}
	}
}

func memorySettingsMenu() {
	clearScreen()
	fmt.Println("\n\033[1;36m内存设置\033[0m")
	fmt.Println("----------------------------------------")
	fmt.Printf("当前默认内存: %dMB\n", config.DefaultMemory)
	fmt.Println("----------------------------------------")
	fmt.Print("请输入新的内存大小(MB): ")

	var mem int
	fmt.Scanln(&mem)

	if mem < 1024 || mem > 32768 {
		fmt.Println("内存大小应在1024-32768MB之间")
		time.Sleep(2 * time.Second)
		return
	}

	config.DefaultMemory = mem
	saveConfig()
	fmt.Printf("默认内存已设置为 %dMB\n", mem)
	time.Sleep(2 * time.Second)
}

func editTranslationDict() {
	clearScreen()
	fmt.Println("\n\033[1;36m日志翻译字典编辑\033[0m")
	fmt.Println("----------------------------------------")
	fmt.Println("当前字典规则:")

	dictPath := filepath.Join(CACHE_DIR, DICT_FILE)
	content, err := os.ReadFile(dictPath)
	if err != nil {
		fmt.Println("无法读取字典文件:", err)
	} else {
		fmt.Println(string(content))
	}

	fmt.Println("\n操作选项:")
	fmt.Println("1. 使用系统编辑器编辑")
	fmt.Println("2. 恢复默认字典")
	fmt.Println("0. 返回主菜单")
	fmt.Println("----------------------------------------")
	fmt.Print("请选择操作: ")

	var choice int
	fmt.Scanln(&choice)

	switch choice {
	case 0:
		return
	case 1:
		var editor string
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "nano"
		}

		cmd := exec.Command(editor, dictPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Println("编辑失败:", err)
		} else {
			fmt.Println("字典已更新，重新加载中...")
			loadTranslationDict()
		}
		time.Sleep(2 * time.Second)
	case 2:
		defaultDict := []byte(`Player [a-zA-Z0-9_]+ joined#玩家 $0 加入游戏
Done \(\d+\.\d+s\)!#启动完成 (耗时 $0 秒)
Stopping server#正在停止服务器
Preparing spawn area: (\d+)%#生成出生点区域: $1%`)
		if err := os.WriteFile(dictPath, defaultDict, 0644); err != nil {
			fmt.Println("恢复默认字典失败:", err)
		} else {
			fmt.Println("默认字典已恢复，重新加载中...")
			loadTranslationDict()
		}
		time.Sleep(2 * time.Second)
	}
}

func clearScreen() {
	switch runtime.GOOS {
	case "linux", "darwin":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}
