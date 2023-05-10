package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	NumNodes = 4
	Inf      = 1000000
)

var nodes = make([]*Node, NumNodes)

type Message struct {
	Src  int
	Lens []int
}

type Node struct {
	Id        int
	Cost      []int
	Shortcuts []int
	Next      []int
	Messages  chan Message
	Done      chan bool
}

func NewNode(id int, nxt, short, cost []int) *Node {
	return &Node{
		Id:        id,
		Cost:      cost,
		Next:      nxt,
		Shortcuts: short,
		Messages:  make(chan Message, 100),
		Done:      make(chan bool, 1),
	}
}

func (n *Node) Run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.Done:
			return
		case <-ticker.C:
			n.SendCostVector()
		case m := <-n.Messages:
			n.UpdateCostVector(m)
		}
	}
}

func (n *Node) SendCostVector() {
	for neighbor := 0; neighbor < NumNodes; neighbor++ {
		m := Message{
			Src:  n.Id,
			Lens: n.Cost,
		}
		nodes[neighbor].Messages <- m
	}
}

func (n *Node) UpdateCostVector(m Message) {
	for i, mid := range m.Lens {
		if n.Shortcuts[i] < n.Cost[i] {
			n.Next[i] = i
			n.Cost[i] = n.Shortcuts[i]
			for neighbor := 0; neighbor < NumNodes; neighbor++ {
				if neighbor != m.Src && neighbor != n.Id {
					m := Message{
						Src:  n.Id,
						Lens: n.Cost,
					}
					nodes[neighbor].Messages <- m
				}
			}
		}
		oldCost := n.Cost[i]
		newLen := n.Cost[m.Src] + mid
		if oldCost > newLen || (oldCost < newLen && n.Next[i] == m.Src) {
			n.Next[i] = m.Src
			n.Cost[i] = newLen
			for neighbor := 0; neighbor < NumNodes; neighbor++ {
				if neighbor != m.Src && neighbor != n.Id {
					m := Message{
						Src:  n.Id,
						Lens: n.Cost,
					}
					nodes[neighbor].Messages <- m
				}
			}
		}
	}
}

func main() {
	// Initialize nodes
	for i := 0; i < NumNodes; i++ {
		cost := make([]int, NumNodes)
		for j := 0; j < NumNodes; j++ {
			cost[j] = Inf
		}
		cost[i] = 0

		nxt := make([]int, NumNodes)
		short := make([]int, NumNodes)
		if i == 0 {
			short = []int{0, 1, 3, 7}
			nxt = []int{0, 1, 2, 3}
		}
		if i == 1 {
			short = []int{1, 0, 1, Inf}
			nxt = []int{0, 1, 2, -1}
		}
		if i == 2 {
			short = []int{3, 1, 0, 2}
			nxt = []int{0, 1, 2, 3}
		}
		if i == 3 {
			short = []int{7, Inf, 2, 0}
			nxt = []int{0, -1, 2, 3}
		}

		nodes[i] = NewNode(i, nxt, short, cost)
		go nodes[i].Run()
	}

	// Set initial cost vectors
	nodes[0].Cost[1] = 1
	nodes[0].Cost[2] = 3
	nodes[0].Cost[3] = 7
	nodes[1].Cost[0] = 1
	nodes[1].Cost[2] = 1
	nodes[2].Cost[0] = 3
	nodes[2].Cost[1] = 1
	nodes[2].Cost[3] = 2
	nodes[3].Cost[0] = 7
	nodes[3].Cost[2] = 2

	reader := bufio.NewReader(os.Stdin)

	for {
		str, _ := reader.ReadString('\n')
		str = str[:len(str)-1]
		split := strings.Split(str, " ")

		if split[0] == "exit" {
			// Stop nodes
			for _, node := range nodes {
				node.Done <- true
			}
			os.Exit(0)
		} else if split[0] == "show" {
			// Print final cost vectors
			fmt.Printf("\n")
			fmt.Println("Final cost vectors (by far):")
			for _, node := range nodes {
				fmt.Printf("Node %d: %v\n", node.Id, node.Cost)
			}
			fmt.Printf("\n")
		} else if split[0] == "update" {
			a, _ := strconv.Atoi(split[1])
			b, _ := strconv.Atoi(split[2])
			l, _ := strconv.Atoi(split[3])

			nodes[a].Cost[b] = l
			nodes[b].Cost[a] = l
			nodes[a].Shortcuts[b] = l
			nodes[b].Shortcuts[a] = l

			fmt.Printf("\n")
		} else {
			fmt.Printf("\n")
			fmt.Printf("List of supported commands:\n" +
				"exit\t:\tsafe exit programm\n" +
				"show\t:\tprint cost vectors at the moment\n" +
				"update\t:\tupdate edge len. Format: update a b len\n\n")
		}
	}
}
