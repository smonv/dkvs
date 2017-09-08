package client

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

var leader string

func main() {
	var servers []string
	var cluster string
	var set bool
	var get bool
	var key string
	var value string

	flag.StringVar(&cluster, "c", "", "cluster ip address")
	flag.BoolVar(&set, "s", false, "set value for a key")
	flag.BoolVar(&get, "g", false, "get value of a key")
	flag.StringVar(&key, "k", "", "key (only use with flag -s)")
	flag.StringVar(&value, "v", "", "value (only use with flag -s)")

	flag.Parse()

	if len(cluster) <= 0 {
		cluster = os.Getenv("CLUSTER")
	}

	if len(cluster) > 0 {
		members := strings.Split(cluster, ",")
		if len(members) > 0 {
			servers = members
		} else {
			fmt.Println("No member address found")
			return
		}
	} else {
		fmt.Println("Please define cluster members address")
		return
	}

	if set {
		if len(key) > 0 && len(value) > 0 {
			setValue(servers, key, value)
			return
		}
		fmt.Println("Please enter both key(-k) and value(-v)")
	}

	if get {
		if len(key) > 0 {
			var result string
			result = getValue(servers, key)
			fmt.Println(result)
			return
		}
		fmt.Println("Please enter key (-k)")
	}
}

func setValue(servers []string, key string, value string) {
	var target string
	if len(leader) > 0 {
		target = leader
	} else {
		target = randomServer(servers)
	}

	url := fmt.Sprintf("http://%s/store/%s", target, key)

	client := newClient()
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(value)))

	resp, err := client.Do(req)
	if err != nil {
		setValue(servers, key, value)
	} else {
		if resp.ContentLength > 0 {
			body, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			leader = string(body)
			setValue(servers, key, value)
		}
	}

	return
}

func getValue(servers []string, key string) string {
	var result string
	var target string
	if len(leader) > 0 {
		target = leader
	} else {
		target = randomServer(servers)
	}

	url := fmt.Sprintf("http://%s/store/%s", target, key)
	client := newClient()

	req, _ := http.NewRequest("GET", url, nil)

	resp, err := client.Do(req)

	if err != nil {
		result = getValue(servers, key)
	} else {
		if resp.ContentLength > 0 {
			body, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			temp := string(body)
			if responseIsLeaderAddress(temp, servers) {
				leader = temp
				result = getValue(servers, key)
			} else {
				return temp
			}
		}
	}

	return result
}

func randomServer(servers []string) string {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	count := len(servers)
	e := r.Intn(count)
	return servers[e]
}

func newClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
	}
}

func responseIsLeaderAddress(resp string, servers []string) bool {
	for _, server := range servers {
		if server == resp {
			return true
		}
	}
	return false
}
