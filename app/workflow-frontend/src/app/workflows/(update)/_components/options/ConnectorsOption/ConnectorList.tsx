import OptionButton from "@/components/buttons/OptionButton"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import ConnectorService from "@/services/connectors/connectors"
import { ConnectorInfo } from "@/services/connectors/connectors.schema"
import { useQuery } from "@tanstack/react-query"
import { Inspect } from "lucide-react"
import { WorkflowOperationContext } from "../../../_providers/WorkflowOperationProvider"
import { useContext } from "react"

const ConnectorList: React.FC<{ setConnector: React.Dispatch<React.SetStateAction<ConnectorInfo | null>> }> = ({ setConnector }) => {
  const {connectorQuery} = useContext(WorkflowOperationContext)
  if (connectorQuery == null) {
    return
  }
  return (
    <div className='flex-1 flex flex-col gap-3 p-3'>
      <div className='flex flex-col gap-2'>
        <Label className="font-normal">
          Step Name
        </Label>
        <Input />
      </div>

      <ul>
        {connectorQuery.data && connectorQuery.data.map(con => (
          <li key={con.id}>
            <OptionButton
              key={`select-task-option-`}
              Icon={Inspect}
              buttonClass='h-[64px] w-full'
              onClick={() => setConnector(con)}>
              <div className='flex-1 flex gap-2 items-center h-full'>
                <Label className='capitalize text'>{con.name}</Label>
              </div>
            </OptionButton>
          </li>

        ))}


      </ul>
    </div>
  )
}

export default ConnectorList