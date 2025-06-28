import { Button } from '@/components/ui/button'
import React, { useContext } from 'react'
import { MoveLeft } from 'lucide-react'

const OperationLayout: React.FC<{
  children: React.ReactNode,
  backHandler: () => void
}> = ({ children, backHandler }) => {

  return (
    <>
      <div className="p-3 pb-0">
        <Button variant="outline" onClick={backHandler}>
          <MoveLeft />
          Back
        </Button>
      </div>
      {children}
    </>

  )
}

export default OperationLayout