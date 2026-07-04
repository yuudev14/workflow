import PlaybookService from "@/services/playbooks/playbooks";
import { useMutation } from "@tanstack/react-query";
import { toast } from "./use-toast";

const usePlaybookTrigger = ({ playbookId }: { playbookId: string }) => {
  const triggerWorfklowMutation = useMutation({
    mutationFn: async (id: string) => {
      return await PlaybookService.triggerPlaybook(id);
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
  const triggerPlaybookHandler = () => {
    triggerWorfklowMutation.mutate(playbookId);
  };

  return {
    triggerWorfklowMutation,
    triggerPlaybookHandler,
  };
};

export default usePlaybookTrigger;
