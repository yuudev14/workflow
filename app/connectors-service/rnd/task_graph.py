from graphlib import TopologicalSorter

def invert_graph(successor_graph: dict):
    inverted = {}

    all_nodes = set(successor_graph.keys())

    for val in successor_graph.values():
        all_nodes.update(val)

    for node in all_nodes:
        inverted[node] = set()

    for node, targets in successor_graph.items():
        for target in targets:
            inverted[target].add(node)

    print(inverted)

    return inverted

successor_graph = {
    "A": {"B", "C"},
    "B": {"D", "E"},
    "C": {"F"},
    "F": {"G"},
    # "G": {"E"}

}

skip_nodes = set()

predecessor_graph = invert_graph(successor_graph)
ts = TopologicalSorter(predecessor_graph)
ts.prepare()

while ts.is_active():
    ready_nodes = ts.get_ready()
    print(ready_nodes)
    for node in ready_nodes:
        if node in skip_nodes:
            ts.done(node)
            continue
        print(f"Processing {node}")
        if node == "B":
            if 1 == 1:
                skip_nodes.add("E")  
        ts.done(node)
