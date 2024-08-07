package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

var (
	configFilePath         = "config.yaml"
	proxySourceURLs        = getEnvSlice("PROXY_SOURCE_URLS", "https://github.com/aiboboxx/clashfree/raw/main/clash.yml")
	proxySourceNum         = getEnvInt("PROXY_SOURCE_NUM", 200)
	proxyApplyGroups       = getEnvSlice("PROXY_APPLY_GROUPS", "自动选择,节点选择") // 根据自己的模板来
	proxyTypeFilter        = getEnvSlice("PROXY_TYPE_FILTER", "vmess,vless,trojan")
	proxyNameExcludeFilter = getEnvSlice("PROXY_NAME_EXCLUDE_FILTER", ".*AD,.*机场") // 过滤广告
	proxyNameFilter        = getEnvSlice("PROXY_NAME_FILTER", ".*")
	excludePatterns        []*regexp.Regexp
	includePatterns        []*regexp.Regexp
)

type ConfigData struct {
	URLs         []string `json:"urls"`
	TemplateFile string   `json:"template_file"`
	OutputFile   string   `json:"output_file"`
}

func init() {
	excludePatterns = compilePatterns(proxyNameExcludeFilter)
	includePatterns = compilePatterns(proxyNameFilter)
	fmt.Println("All Envs:")
	fmt.Printf("PROXY_SOURCE_URLS: %v\n", proxySourceURLs)
	fmt.Printf("PROXY_SOURCE_NUM: %d\n", proxySourceNum)
	fmt.Printf("PROXY_APPLY_GROUPS: %v\n", proxyApplyGroups)
	fmt.Printf("PROXY_TYPE_FILTER: %v\n", proxyTypeFilter)
	fmt.Printf("PROXY_NAME_EXCLUDE_FILTER: %v\n", proxyNameExcludeFilter)
}

func main() {

	app := fiber.New()
	// 启动定时任务
	c := cron.New()
	c.AddFunc("@hourly", scheduledJob)

	c.Start()

	// 立即执行一次生成配置文件

	scheduledJob()
	app.Get("/", healthCheck)
	app.Get("/config.yaml", getConfig)
	log.Fatal(app.Listen(":8000"))
}

func healthCheck(c fiber.Ctx) error {
	return c.SendString("ok")
}

func getConfig(c fiber.Ctx) error {

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).SendString("config.yaml not found")
	}
	return c.SendFile(configFilePath)
}

func scheduledJob() {

	config := ConfigData{
		URLs:         proxySourceURLs,
		TemplateFile: "template.yaml",
		OutputFile:   configFilePath,
	}
	generateConfig(config.URLs, config.TemplateFile, config.OutputFile)
}
func generateConfig(urls []string, templateFile, outputFile string) {

	var filteredProxies []map[string]interface{}

	for _, url := range urls {
		yamlData := loadYAMLFromURL(url)
		var data map[string]interface{}
		err := yaml.Unmarshal([]byte(yamlData), &data)
		// data := make(map[string]interface{})
		// err := yaml.Unmarshal([]byte(yamlData), &data)
		if err != nil {
			log.Printf("Error unmarshaling YAML from %s: %v", url, err)
			continue
		}
		p := filterProxies(data)
		filteredProxies = append(filteredProxies, p...)
	}
	if len(filteredProxies) == 0 {
		log.Printf("No proxies!!! URLs: %v", urls)
		return
	}
	template := loadTemplateFromFile(templateFile)
	updatedTemplate := replaceTemplateProxies(template, filteredProxies)
	saveYAMLToFile(updatedTemplate, outputFile)
}

func loadYAMLFromURL(url string) string {

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error fetching YAML data: %v", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	return string(body)
}

func filterProxies(data map[string]interface{}) []map[string]interface{} {

	var filteredProxies []map[string]interface{}
	proxies, ok := data["proxies"].([]interface{})
	if !ok {
		return filteredProxies
	}

	for _, proxy := range proxies {
		proxyMap, ok := proxy.(map[string]interface{})
		if !ok {
			continue
		}

		proxyType, _ := proxyMap["type"].(string)
		proxyName, _ := proxyMap["name"].(string)
		if !contains(proxyTypeFilter, proxyType) {
			continue
		}

		if matchesAny(excludePatterns, proxyName) {
			fmt.Printf("Filtered (excluded pattern matched): name=%s\n", proxyName)
			continue
		}

		if len(includePatterns) > 0 {
			if matchesAny(includePatterns, proxyName) {
				filteredProxies = append(filteredProxies, proxyMap)
				fmt.Printf("Included (pattern matched): name=%s\n", proxyName)
			}

		} else {

			if len(filteredProxies) < proxySourceNum {
				filteredProxies = append(filteredProxies, proxyMap)
			} else {
				fmt.Printf("Filtered (count limit reached): name=%s\n", proxyName)
			}

		}

	}

	fmt.Printf("Filtered proxies:\n%v\n", filteredProxies)
	return filteredProxies
}

func loadTemplateFromFile(templateFile string) map[string]interface{} {

	data, err := ioutil.ReadFile(templateFile)

	if err != nil {
		log.Fatalf("Error reading template file: %v", err)
	}

	var template map[string]interface{}
	err = yaml.Unmarshal(data, &template)

	if err != nil {
		log.Fatalf("Error unmarshaling template YAML: %v", err)
	}

	return template
}

func replaceTemplateProxies(template map[string]interface{}, proxies []map[string]interface{}) map[string]interface{} {

	seenNames := make(map[string]bool)
	var proxyNames []string
	for i, proxy := range proxies {
		name, ok := proxy["name"].(string)
		if !ok {
			// 如果 name 不是字符串，我们生成一个新的名字
			name = fmt.Sprintf("proxy-%d", i)
			proxy["name"] = name
		}

		if seenNames[name] {
			newName := fmt.Sprintf("%s-%s", name, generateRandomString(8))
			for seenNames[newName] {
				newName = fmt.Sprintf("%s-%s", name, generateRandomString(8))
			}
			proxy["name"] = newName
			name = newName
		}
		seenNames[name] = true
		proxyNames = append(proxyNames, name)
		proxies[i] = proxy
	}

	template["proxies"] = proxies

	if groups, ok := template["proxy-groups"].([]interface{}); ok {
		for i, group := range groups {
			if g, ok := group.(map[string]interface{}); ok {
				if groupName, ok := g["name"].(string); ok && contains(proxyApplyGroups, groupName) {
					g["proxies"] = proxyNames
					groups[i] = g
				}
			}
		}
		template["proxy-groups"] = groups
	}

	return template
}

func saveYAMLToFile(data map[string]interface{}, filename string) {

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		log.Fatalf("Error marshaling YAML: %v", err)
	}

	err = ioutil.WriteFile(filename, yamlData, 0644)
	if err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}
	fmt.Printf("Filtered proxies saved to %s\n", filename)
}

func generateRandomString(length int) string {

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}

	return string(b)
}

func getEnvSlice(key, defaultValue string) []string {

	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}

	return strings.Split(value, ",")
}

func getEnvInt(key string, defaultValue int) int {

	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func compilePatterns(patterns []string) []*regexp.Regexp {

	var compiledPatterns []*regexp.Regexp

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err == nil {
			compiledPatterns = append(compiledPatterns, re)
		}
	}

	return compiledPatterns
}

func contains(slice []string, item string) bool {

	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func matchesAny(patterns []*regexp.Regexp, s string) bool {

	for _, pattern := range patterns {
		if pattern.MatchString(s) {
			return true
		}
	}
	return false
}
