package utils

type WorkflowData struct {
	Graph        map[string][]string `json:"graph"`
	CurrentNode  string              `json:"current_node"`
	CurrentQueue []string            `json:"current_queue"`
	Visited      []string            `json:"visited"`
}

func IsAcyclicGraph(graph map[string][]string) bool {
	visit := make(Set[string])

	var dfs func(node string) bool
	dfs = func(node string) bool {
		if visit.Has(node) {
			return true
		}
		if len(graph[node]) == 0 {
			return false
		}

		visit.Add(node)

		for _, neighbor := range graph[node] {
			if dfs(neighbor) {
				return true
			}
		}
		visit.Remove(node)
		graph[node] = []string{}

		return false
	}

	for node := range graph {
		if dfs(node) {
			return true
		}
	}

	return false

}
