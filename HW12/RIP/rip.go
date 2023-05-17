package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

const (
	INF     = 100
	maxHops = 15
)

type Path struct {
	Cost    int
	NextHop string
}

type Router struct {
	IP          string
	RoutingTbl  sync.Map
	Connections []string
}

type AS struct {
	Routers []Router
}

func main() {
	as, err := loadASConfig("as.json")
	if err != nil {
		log.Fatalf("Failed to load AS config: %v", err)
	}

	initializeRoutingTables(&as)

	var wg sync.WaitGroup
	wg.Add(len(as.Routers))

	for i := range as.Routers {
		go func(routerIndex int) {
			defer wg.Done()
			router := &as.Routers[routerIndex]
			updateRoutingTable(router, &as)
			fmt.Printf("Current iteration\n"+
				"%s\n"+
				"---------------------------------------------------------------\n",
				displayRoutingTables(&as))
		}(i)
	}

	wg.Wait()

	fmt.Println("RIP protocol finished")
	fmt.Println(displayRoutingTables(&as))
}

func loadASConfig(filename string) (AS, error) {
	file, err := os.Open(filename)
	if err != nil {
		return AS{}, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return AS{}, err
	}

	var as AS
	err = json.Unmarshal(data, &as)
	if err != nil {
		return AS{}, err
	}

	return as, nil
}

func initializeRoutingTables(as *AS) {
	for i := range as.Routers {
		router := &as.Routers[i]

		for j := range as.Routers {
			if i != j {
				router.RoutingTbl.Store(as.Routers[j].IP, Path{
					Cost:    INF,
					NextHop: as.Routers[i].IP,
				})
			}
		}

		for _, neighborIP := range router.Connections {
			router.RoutingTbl.Store(neighborIP, Path{
				Cost:    1,
				NextHop: neighborIP,
			})
		}
	}
}

func updateRoutingTables(as *AS) {
	updated := true
	iteration := 0
	for updated {
		updated = false

		for i := range as.Routers {
			router := &as.Routers[i]
			updated = updated || updateRoutingTable(router, as)
		}

		iteration++
		fmt.Printf("Iteration %d:\n", iteration)
		fmt.Printf("%s", displayRoutingTables(as))
		fmt.Println()
	}
}

func updateRoutingTable(router *Router, as *AS) bool {
	updated := false

	for _, neighborIP := range router.Connections {
		neighbor := findRouterByIP(as, neighborIP)

		if neighbor == nil {
			continue
		}

		neighbor.RoutingTbl.Range(func(key, value interface{}) bool {
			k := key.(string)
			v := value.(Path)
			currValRaw, ok := router.RoutingTbl.Load(k)
			if !ok {
				return true
			}
			currVal := currValRaw.(Path)
			if currVal.Cost == INF || currVal.Cost > v.Cost+1 {
				router.RoutingTbl.Store(k, Path{
					Cost:    v.Cost + 1,
					NextHop: neighborIP,
				})
				updated = true
			}
			return true
		})
		/*for destIP, path := range neighbor.RoutingTbl {
			if router.RoutingTbl[destIP].Cost == INF || router.RoutingTbl[destIP].Cost > path.Cost+1 {
				router.RoutingTbl[destIP] = Path{
					Cost:    path.Cost + 1,
					NextHop: neighborIP,
				}
				updated = true
			}
		}*/
	}

	return updated
}

func displayRoutingTables(as *AS) string {
	ret := ""
	for i := range as.Routers {
		ret += fmt.Sprintf("Router: %s\n", as.Routers[i].IP)
		ret += fmt.Sprintf("Destination IP\t\tNext Hop\t\tCost\n")

		as.Routers[i].RoutingTbl.Range(func(key, value interface{}) bool {
			k := key.(string)
			v := value.(Path)
			if v.Cost <= maxHops {
				ret += fmt.Sprintf("%s\t\t%s\t\t%d\n", k, v.NextHop, v.Cost)
			}
			return true
		})

		/*for destIP, path := range router.RoutingTbl {
			ret += fmt.Sprintf("%s\t\t%s\t\t%d\n", destIP, path.NextHop, path.Cost)
		}*/

		ret += "\n"
	}
	return ret
}

func findRouterByIP(as *AS, ip string) *Router {
	for i := range as.Routers {
		if as.Routers[i].IP == ip {
			return &as.Routers[i]
		}
	}
	return nil
}
