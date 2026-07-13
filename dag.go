package main

import (
	"strings"
)

// K is the the key used to reference node V
type Graph struct {
	nodes    map[int]Change
	edges    map[int][]int
	indegree map[int]int
}

func NewGraph(nodes []Change) *Graph {
	var (
		nodeMap  = map[int]Change{}
		indegree = map[int]int{}
	)

	// set defaults for the graph
	for _, n := range nodes {
		nodeMap[n.ID()] = n
		indegree[n.ID()] = 0
	}

	g := &Graph{
		nodes:    nodeMap,
		edges:    map[int][]int{},
		indegree: indegree,
	}

	g.computeEdges()

	return g
}

func (g *Graph) computeEdges() {

	for ID, node := range g.nodes {
		// look at each node and check to see if
		// any node depends on it to resolve
	DEPS_LOOP:
		for prospectID, prospect := range g.nodes {
			if !g.isDependant(node, prospect) {
				continue DEPS_LOOP
			}

			// pretty sure that the prospect is a dependancy
			g.edges[prospectID] = append(g.edges[prospectID], ID)
			g.indegree[ID] += 1
		}
	}
}

// returns true if the graph is acyclic
func (g *Graph) OutputChanges() ([]Change, bool) {
	var (
		queue = []Change{}
		out   = []Change{}
	)

	// build queue of free nodes
	for id, e := range g.nodes {
		if g.indegree[id] == 0 {
			queue = append(queue, e)
		}
	}

	// Loop until queue is empty. Khans algorithm.
	//
	//	1. Pop an item from the queue and append to output
	//	2. Pop a node from the queue, add it to out, and decrement indegree for every dependant of the node
	//	3. Add every node that has indegree==0 to the queue.
	//	4. Repeat until queue is empty.

	// the graph is asyclic if the number of output nodes
	// equal the total number of nodes

	for len(queue) > 0 {

		// shift the next node in the queue.
		node := queue[0]
		queue = queue[1:]

		out = append(out, node)

		for _, dependantID := range g.edges[node.ID()] {
			// from node to its dependants
			g.indegree[dependantID]--

			// if indegree for dependant is zero, we append the node to the queue.
			if g.indegree[dependantID] == 0 {
				queue = append(queue, g.nodes[dependantID])
			}
		}
	}

	isAsyclic := len(g.nodes) == len(out)

	return out, isAsyclic
}

func (g *Graph) isDependant(node, dependancy Change) bool {
	// can't depend on it self
	if node.ID() == dependancy.ID() {
		return false
	}

	// cheap check to see if the node is possibly a dependancy
	// (meaning that the dependancy entry is nested inside node)
	if !strings.HasPrefix(dependancy.From.Path, node.From.Path) {
		return false
	}

	// count number of slashes. if they are equal, it means that
	// that the entries are siblings, and don't depend on eachother.
	if strings.Count(node.From.Path, "/") == strings.Count(dependancy.From.Path, "/") {
		return false
	}

	return true
}
