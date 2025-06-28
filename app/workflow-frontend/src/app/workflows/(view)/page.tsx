"use client"

import React from 'react'
import CreateWorkflowForm from '../_components/CreateWorkflowForm'

export default function Page() {

  
  return (
    <div>
      <div className="py-3 px-5 flex justify-between items-center h-16">
        <div className="flex gap-2 ml-auto">
          <CreateWorkflowForm />
          
        </div>
      </div>
    </div>
  )
}
