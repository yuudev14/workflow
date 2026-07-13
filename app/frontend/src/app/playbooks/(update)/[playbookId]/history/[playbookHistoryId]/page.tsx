"use client";
import React from "react";
import RunHistoryDetail from "@/components/executions/RunHistoryDetail";

const Page: React.FC<{ params: Promise<{ playbookHistoryId: string }> }> = ({
  params,
}) => {
  const { playbookHistoryId } = React.use(params);
  return <RunHistoryDetail playbookHistoryId={playbookHistoryId} />;
};

export default Page;
