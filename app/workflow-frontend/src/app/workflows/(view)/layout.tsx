"use client";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import React from "react";

const Layout = ({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) => {
  return (
    <div className="flex justify-center h-full">
      <div className="flex flex-col flex-1 gap-10 px-2 max-w-7xl py-15">
        <div className="flex flex-col gap-1">
          <h2 className="font-bold">Overview</h2>
          <p className="text-muted-foreground">
            All the workflows, credentials and executions you have access to
          </p>
        </div>
        <div className="flex">
          {[...Array(5)].map((k, i) => (
            <button
              key={`worfklow-graph-${i}`}
              className={`relative z-30 flex flex-col justify-center flex-1 gap-1 px-6 py-4 text-left border-t bg-muted/50 hover:bg-muted/30 even:border-l sm:border-t-0 sm:border-l sm:px-8 sm:py-6 ${
                i === 4 ? " border-r" : ""
              }`}>
              <span className="text-xs text-muted-foreground">
                Success Execution
              </span>
              <span className="text-lg font-bold leading-none sm:text-3xl">
                5,444
              </span>
            </button>
          ))}
        </div>
        <div>
          <Tabs defaultValue="account">
            <TabsList>
              <TabsTrigger value="worflows">Workflows</TabsTrigger>
              <TabsTrigger value="executions">Executions</TabsTrigger>
            </TabsList>
            <TabsContent value="account">
              Make changes to your account here.
            </TabsContent>
            <TabsContent value="password">
              Change your password here.
            </TabsContent>
          </Tabs>
        </div>
        {children}
      </div>
    </div>
  );
};

export default Layout;
