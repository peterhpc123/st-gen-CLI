package main

import (
	"fmt"
	"github.com/beevik/etree"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

const LOCAL = "${user.home}/.m2/settings.xml"

const REMOTE = "https://github.corp.ebay.com/RaptorTeam/Maven/blob/master/Raptor2/LOCAL/settings.xml"

const API = "https://artifactory.qa.ebay.com/artifactory/api/security/apiKey"

func main() {
	if err := NewLoginCommand().Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

type loginOption struct {
	//identityEndpoint string
	username string
	password string
}

func (op *loginOption) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&op.username, "username", "", "please input username(NT):")
	fs.StringVar(&op.password, "password", "", "please input password:")
}
func NewLoginCommand() *cobra.Command {
	option := &loginOption{}
	cmd := &cobra.Command{
		Use:     "login",
		Short:   "Login to st-gen",
		Long:    "Login to st-gen",
		Example: "",
		Run: func(cmd *cobra.Command, args []string) {
			if exist, _ := PathExists(LOCAL); exist { //存在直接更新maven
				fmt.Println("Found settings.xml at LOCAL dir!")
			} else {
				//下载settings.xml文件
				resp, err := http.Get(REMOTE)
				if err != nil {
					panic(err)
				}
				defer resp.Body.Close()
				//create output file
				out, err := os.Create(LOCAL)
				if err != nil {
					panic(err)
				}
				defer out.Close()
				//copy data
				_, err = io.Copy(out, resp.Body)
				if err != nil {
					panic(err)
				}
				//login and get yubiKey
				//username,yubiKey
				yubiKey, err := runLogin(option)
				if err != nil {
					log.Fatal("error when login, username/password is not  provided!")
				}
				//更新setting.xml文件
				doc := etree.NewDocument()
				if err := doc.ReadFromFile(LOCAL); err != nil {
					panic(err)
				}
				root := doc.SelectElement("settings")
				fmt.Println("Root element:", root.Tag)
				for _, server := range root.SelectElements("server") {
					uname := server.SelectElement("username")
					uname.SetText(option.username) //uname为login的username
					password := server.SelectElement("password")
					password.SetText(yubiKey) //password为login的yubiKey
				}
				fmt.Println("updated settings.xml at LOCAL dir, you can use it update all ebaycentral repository IDs!")
			}
		},
	}
	option.addFlags(cmd.Flags())
	return cmd
}
func runLogin(option *loginOption) (string, error) {
	//if option.username == "" || option.password == "" {
	//	//errors.New("username/password is not provided")
	//	return ""
	//}
	//password, err := getPasswordFromStdin()
	//if err != nil {
	//	return err
	//}
	//api call
	yubiKey, err := authenticate(option.username, option.password, API)
	return yubiKey, err
}

//func getPasswordFromStdin() (string, error) {
//	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
//	if err != nil {
//		return "", fmt.Errorf("unable to get user password:#{err}")
//	}
//	password := string(bytePassword)
//	password = strings.TrimSpace(password)
//	return password, nil
//}
func authenticate(username, password, identityEndpoint string) (string, error) { //api call
	curl := exec.Command("curl", "-X", "POST", "-u", fmt.Sprintf("%s:%s", username, password), identityEndpoint)
	out, err := curl.Output()
	if err != nil {
		return "wrong username/password, please retry.", err
	}
	return string(out), nil
}
