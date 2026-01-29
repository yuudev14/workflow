import { Skeleton } from "@/components/ui/skeleton";

const WorkflowListLoader = () => {
  return [...Array(5)].map((_, _i) => (
    <li key={`workflow-list-skeleton-${_i}`}>
      <Skeleton className="h-[81.33px] rounded-sm"></Skeleton>
    </li>
  ));
};

export default WorkflowListLoader