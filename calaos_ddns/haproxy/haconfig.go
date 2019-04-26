package haproxy

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"text/template"
)

type Config struct {
	Banner         string
	Backends       []*HaBackend
	DefaultBackend string
}

type HaBackend struct {
	Name       string //Haproxy backend name
	Fqdn       string //Fqdn for the SNI check
	ServerName string //Server name in the backend
	Ip         string //Ip of the server
	Port       string //Port of the server
	Default    bool   //true if this is the default backend
}

var (
	Default = &Config{
		Banner: "### DO NOT EDIT. File automatically generated by calaos_ddns",
	}
)

func createConfig() *Config {
	c := Default

	//Create default calaos-server backend
	defBackend := &HaBackend{
		ServerName: "calaos-server",
		Ip:         "127.0.0.1",
		Port:       "5454",
		Default:    true,
	}
	c.Backends = append(c.Backends, defBackend)

	return c
}

func ParseDomains(domain string, subdomains []string) (*Config, error) {
	config := createConfig()

	maindomain, ip, port, err := parseOpt(domain, false)
	if err != nil {
		return nil, fmt.Errorf("Failure to parse %v", domain)
	}

	config.Backends[0].Name = maindomain
	config.DefaultBackend = maindomain

	//Another ip for calaos-server can be set
	if ip != "" {
		config.Backends[0].Ip = ip
		config.Backends[0].Port = port
	}

	for _, sub := range subdomains {
		name, ip, port, err := parseOpt(sub, true)
		if err != nil {
			return nil, fmt.Errorf("Failure to parse %v: %v", sub, err)
		}

		habackend := &HaBackend{
			Name:       name,
			Fqdn:       name + "." + maindomain + ".calaos.fr",
			ServerName: name + "-server",
			Ip:         ip,
			Port:       port,
			Default:    false,
		}

		config.Backends = append(config.Backends, habackend)
	}

	return config, nil
}

func parseOpt(opt string, sub bool) (name, ip, port string, err error) {
	if i := strings.IndexByte(opt, '='); i >= 0 {
		name = opt[:i]
		rest := opt[i+1:]

		if i = strings.IndexByte(rest, ':'); i >= 0 {
			ip = rest[:i]
			port = rest[i+1:]
		}

		if ip == "" || port == "" {
			err = errors.New("No ip/port defined")
			return
		}
	} else {
		name = opt
	}

	if name == "" {
		err = errors.New("no hostname defined")
	}

	if sub {
		if !isValidSubHostname(name) {
			err = errors.New("hostname is invalid")
		}
	} else {
		if !isValidHostname(name) {
			err = errors.New("hostname is invalid")
		}
	}

	return
}

func RenderConfig(outFile string, templateFile string, config *Config) error {
	f, err := ioutil.ReadFile(templateFile)
	if err != nil {
		return err
	}

	fp, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer fp.Close()

	t := template.Must(template.New(templateFile).Parse(string(f)))
	err = t.Execute(fp, &config)
	if err != nil {
		return err
	}

	return nil
}

func isValidHostname(host string) bool {
	valid, _ := regexp.Match("^[a-z0-9]{4,32}$", []byte(host))

	return valid
}

func isValidSubHostname(host string) bool {
	valid, _ := regexp.Match("^[a-z0-9]{2,32}$", []byte(host))

	return valid
}
