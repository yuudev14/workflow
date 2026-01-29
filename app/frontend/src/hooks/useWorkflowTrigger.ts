import WorkflowService from "@/services/worfklows/workflows";
import { useMutation } from "@tanstack/react-query";
import { toast } from "./use-toast";

const useWorkflowTrigger = ({ workflowId }: { workflowId: string }) => {
  const triggerWorfklowMutation = useMutation({
    mutationFn: async (id: string) => {
      return await WorkflowService.triggerWorkflow(id);
    },
    onSuccess: () => {
      toast({
        title: "succesfully triggered the workflow",
      });
    },
    onError(error) {
      toast({
        title: "Error when triggering the workflow",
        description: error.message,
      });
    },
  });

  /**
   * This triggers the workflow for testing purpose
   */
  const triggerWorkflowHandler = () => {
    triggerWorfklowMutation.mutate(workflowId);
  };

  return {
    triggerWorfklowMutation,
    triggerWorkflowHandler,
  };
};

export default useWorkflowTrigger;
