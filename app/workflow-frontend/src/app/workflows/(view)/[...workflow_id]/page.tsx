"use client"

import ReactFlowPlayground from '@/components/react-flow/ReactFlowPlayground'
import { Button } from '@/components/ui/button'
import WorkflowService from '@/services/worfklows/workflows'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { useQuery } from '@tanstack/react-query'
import { Node } from '@xyflow/react'
import React, { useEffect, useMemo, useState } from 'react'
import { Label } from '@/components/ui/label'
import { Tasks } from '@/services/worfklows/workflows.schema'
import { PlaybookTaskNode } from '@/components/react-flow/schema'
import lib from '@/lib'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion"
import { Input } from '@/components/ui/input'
import Link from 'next/link'
import CreateWorkflowForm from '../../_components/CreateWorkflowForm'




const Page: React.FC<{ params: Promise<{ workflow_id: string }> }> = ({ params }) => {
  const { workflow_id: workflowId } = React.use(params)
  const [isOpenPlaybookInfo, setIsOpenPlaybookInfo] = useState<boolean>(false)
  const [currentNode, setCurrentNode] = useState<Tasks | null | Partial<Tasks> & {
    label?: string
  }>(null)

  const workflowQuery = useQuery({
    queryKey: ['workflow-' + workflowId], queryFn: async () => {
      return WorkflowService.getWorkflowById(workflowId)
    }
  })

  useEffect(() => {
    if (!isOpenPlaybookInfo) setCurrentNode(null)
  }, [isOpenPlaybookInfo])



  const nodes: Node<PlaybookTaskNode>[] = useMemo(() => {
    if (workflowQuery.data === undefined || !Array.isArray(workflowQuery.data.tasks)) return []
    return workflowQuery.data.tasks.map((task) => {
      const data: Node<PlaybookTaskNode> = {
        id: task.id,
        data: task.name === "start" ? { label: "start", ...task } : task,
        position: { x: task.x, y: task.y },
        type: task.name === "start" ? "input" : "playbookNodes",
      }
      return data
    })

  }, [workflowQuery.data])


  const edges = useMemo(() => {
    if (workflowQuery.data === undefined || !Array.isArray(workflowQuery.data.edges)) return []
    return workflowQuery.data.edges.map((edge) => ({
      id: edge.id,
      source: edge.source_id,
      target: edge.destination_id,
    }))

  }, [workflowQuery.data])

  const valuesToRender = useMemo(() => {
    if (currentNode === null) {
      return []
    }
    const keysToRender = [
      "connector_name", "operation", "created_at", "updated_at"
    ]
    return keysToRender.map((value) => {
      if (["created_at", "updated_at"].includes(value)) {
        return {
          label: value.replace(/_/g, " "),
          value: lib.utils.readableDate(currentNode[value as keyof Tasks] as string)
        }
      }
      return {
        label: value.replace("_", " "),
        value: currentNode[value as keyof Tasks]
      }
    }).filter(val => val.value)

  }, [currentNode])


  if (workflowQuery.data === undefined) {
    return
  }

  return (
    <React.Fragment>
      <div className="py-3 px-5 flex justify-between items-center h-16">
        <p className="font-medium text-xl">{workflowQuery.data.name}</p>
        <div className="flex gap-2">
          <Button>Trigger</Button>
          <Button>Delete</Button>
          <Link href={`/workflows/update/${workflowId}`}>
            <Button>Update</Button>
          </Link>
          <CreateWorkflowForm />

        </div>
      </div>
      <div className="h-[calc(100vh-8rem)]">
        <ReactFlowPlayground<PlaybookTaskNode>
          flowProps={{
            defaultNodes: nodes,
            defaultEdges: edges,
            nodesDraggable: false,
            onNodeDoubleClick: (_, node) => {
              console.log(node)
              setCurrentNode(node.data)
              setIsOpenPlaybookInfo(true)
            }
          }} />
      </div>
      {currentNode && (
        <Sheet open={isOpenPlaybookInfo} onOpenChange={setIsOpenPlaybookInfo}>
          <SheetContent side="right" className='min-w-[600px] flex flex-col'>
            <SheetHeader>

              <SheetTitle className='flex flex-col text-2xl'>
                <span className='text-xs text-muted-foreground'>Step Name</span>
                {currentNode.name}
              </SheetTitle>
              <SheetDescription>{currentNode.description}</SheetDescription>
            </SheetHeader>
            <div className="flex flex-1 flex-col gap-7">

              {valuesToRender.map(val => (
                <RenderKeyValue label={val.label} value={val.value as string} key={val.label} />
              ))}

              <Accordion type="multiple" className="w-full">
                <AccordionItem value="config">
                  <AccordionTrigger>Config</AccordionTrigger>
                  <AccordionContent>
                    <Input disabled value="asas" />
                  </AccordionContent>
                </AccordionItem>
                <AccordionItem value="parameters">
                  <AccordionTrigger>Parameters</AccordionTrigger>
                  <AccordionContent className='flex flex-col gap-4'>
                    <div className='flex flex-col gap-2'>
                      <Label className="font-normal">
                        Parameter 1
                      </Label>
                      <Input disabled value="asas" />
                    </div>
                    <div className='flex flex-col gap-2'>
                      <Label className="font-normal">
                        Parameter 1
                      </Label>
                      <Input disabled value="asas" />
                    </div>
                    <div className='flex flex-col gap-2'>
                      <Label className="font-normal">
                        Parameter 1
                      </Label>
                      <Input disabled value="asas" />
                    </div>


                  </AccordionContent>
                </AccordionItem>

              </Accordion>


            </div>
            <SheetFooter>
              <SheetClose asChild>
                <Button type="button">Close</Button>
              </SheetClose>

            </SheetFooter>
          </SheetContent>
        </Sheet>
      )}


    </React.Fragment>
  )
}

const RenderKeyValue: React.FC<{ label: string, value: string }> = ({ label, value }) => {
  return (
    <div className='flex items-center justify-between'>
      <Label className='capitalize'>{label}</Label>
      <p className="bg-secondary px-3 py-1 rounded">{value}</p>
    </div>

  )
}

export default Page