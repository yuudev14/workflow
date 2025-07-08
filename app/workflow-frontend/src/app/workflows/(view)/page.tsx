"use client"

import React from 'react'
import CreateWorkflowForm from '../_components/CreateWorkflowForm'
import { useQuery } from '@tanstack/react-query'
import WorkflowService from '@/services/worfklows/workflows'
import Link from 'next/link'

export default function Page() {
  const workflowQuery = useQuery({
    queryKey: ['workflow-lists'], queryFn: async () => {
      return WorkflowService.getWorkflows()
    }
  })

  if (workflowQuery.data === undefined) {
    return <></>
  }

  
  return (
    <div>
      {/* <div className="py-3 px-5 flex justify-between items-center">
        <div className="flex gap-2">
          <CreateWorkflowForm />
          <ul>
            {workflowQuery.data.entries.map((workflow) => (
              <Link
                href={"/workflows/" + workflow.id}
                key={`playbook-${workflow.id}`}
                className="flex flex-col items-start gap-2 whitespace-nowrap border-b p-4 text-sm leading-tight last:border-b-0 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
              >
                <div className="flex w-full items-center gap-2">
                  <p className="font-medium text-lg">{workflow.name}</p>
                  <p className="ml-auto text-xs">active</p>
                </div>

                <p className="line-clamp-2 w-full whitespace-break-spaces">
                  {workflow.description}
                </p>
              </Link>
            ))}
          </ul>
          
          
        </div>
      </div> */}
    </div>
  )
}
