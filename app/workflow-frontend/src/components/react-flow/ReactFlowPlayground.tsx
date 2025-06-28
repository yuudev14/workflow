import { ReactFlow, Controls, Background, ReactFlowProps, Node, Edge, MiniMap } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { useTheme } from 'next-themes';
import { useMemo } from 'react';
import PlaybookNode from './PlaybookNode';

const nodeTypes = {
  playbookNodes: PlaybookNode
}

const ReactFlowPlayground = <T extends Record<string, any>>({ flowProps }: {
  flowProps: ReactFlowProps<Node<T>, Edge>
}) => {
  const { theme } = useTheme()

  const reactFlowTheme = useMemo(() => {
    return ["dark", "light", "system", undefined].includes(theme) ? theme as "dark" | "light" | "system" | undefined : "system"
  }, [theme])

  return (
    <div className='h-full w-full'>
      <ReactFlow
        colorMode={reactFlowTheme}
        nodeTypes={nodeTypes}
        zoomOnDoubleClick={false}

        {...flowProps}

      >
        <MiniMap />
        <Background />
        <Controls />
      </ReactFlow>
    </div>
  );
}

export default ReactFlowPlayground