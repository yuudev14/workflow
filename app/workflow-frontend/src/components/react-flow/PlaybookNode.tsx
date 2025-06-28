import { useCallback } from 'react';
import { Handle, Node, NodeProps, NodeResizer, Position } from '@xyflow/react';
import { Workflow } from 'lucide-react';
import { Avatar, AvatarFallback } from '../ui/avatar';
import { Tasks } from '@/services/worfklows/workflows.schema';

type NodeComponentProps = Node<Tasks>

const PlaybookNode: React.FC<NodeProps<NodeComponentProps>> = (props) => {

  return (
    <>
      {/* Source Handles */}
      <Handle type="source" position={Position.Top} id="top-handle" />
      <Handle type="source" position={Position.Bottom} id="bottom-handle" />
      <Handle type="source" position={Position.Left} id="left-handle" />
      <Handle type="source" position={Position.Right} id="right-handle" />

      {/* Node Content */}
      <div className='flex w-full items-center gap-3'>
        <Avatar>
          <AvatarFallback><Workflow className='size-6 text-primary' /></AvatarFallback>
        </Avatar>
        <div className='flex-1 mr-9'>
          <p className='font-medium'>{props.data.name}</p>
        </div>
      </div>
      {/* Target Handles */}
      <Handle type="target" position={Position.Top} id="target-top" />
      <Handle type="target" position={Position.Bottom} id="target-bottom" />
      <Handle type="target" position={Position.Left} id="target-left" />
      <Handle type="target" position={Position.Right} id="target-right" />

    </>
  )
}

export default PlaybookNode