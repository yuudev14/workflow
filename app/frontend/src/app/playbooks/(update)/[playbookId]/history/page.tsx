import React from "react";
import { Clock } from "lucide-react";
import { EmptyState } from "@/components/soar";

const Page = () => {
  return (
    <div className="flex flex-1 items-center justify-center p-8">
      <EmptyState
        icon={Clock}
        title="Select a run"
        description="Pick an execution from the list to replay it and inspect each step's output."
        className="max-w-sm"
      />
    </div>
  );
};

export default Page;
