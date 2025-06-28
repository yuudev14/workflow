"use client"

import WorkflowService from '@/services/worfklows/workflows';
import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import React from 'react'

const layout = ({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) => {
  const workflowQuery = useQuery({
    queryKey: ['workflow-lists'], queryFn: async () => {
      return WorkflowService.getWorkflows()
    }
  })

  if (workflowQuery.data === undefined) {
    return <></>
  }
  return (
    <div className="flex">
      <div className="md:flex w-[350px] border-r bg-white dark:bg-sidebar h-[calc(100vh-4rem)] overflow-auto">

        <div className="px-0 w-full">
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
      </div>
      <div className="flex-1">
        {children}

      </div>
    </div>
  )
}

export default layout