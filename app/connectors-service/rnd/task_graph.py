from graphlib import TopologicalSorter
from concurrent.futures import ThreadPoolExecutor, as_completed


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
    "G": {"E"}

}

skip_nodes = set()

predecessor_graph = invert_graph(successor_graph)
ts = TopologicalSorter(predecessor_graph)
ts.prepare()


def process_node(node):
    print(f"Processing {node} in parallel")
    if node == "B":
        if 1 == 1:
            skip_nodes.add("E")
    if node == "F":
        raise Exception("hello")
    return node

with ThreadPoolExecutor() as executor:
    while ts.is_active():
        ready_nodes = ts.get_ready()

        for node in ready_nodes:
          if node in skip_nodes:
              ts.done(node)
              continue
        print("processing", [node for node in ready_nodes if node not in skip_nodes])
        futures = {executor.submit(process_node, node): node for node in ready_nodes if node not in skip_nodes}

        for future in as_completed(futures):
            node_done = futures[future]
            result = future.result()
            ts.done(node_done)

