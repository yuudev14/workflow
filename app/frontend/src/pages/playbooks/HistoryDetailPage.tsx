import { useParams } from "react-router";
import React from "react";
import RunHistoryDetail from "@/components/executions/RunHistoryDetail";

const Page = () => {
  const { playbookHistoryId } = useParams() as { playbookHistoryId: string };
  return <RunHistoryDetail playbookHistoryId={playbookHistoryId} />;
};

export default Page;
