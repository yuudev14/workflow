"use client";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import React from "react";

const Layout = ({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) => {
  return (
    <div className="h-full flex justify-center">
      <div className="max-w-7xl flex-1 px-2 py-5 flex flex-col gap-10">
        <div className="flex flex-col gap-1">
          <h2>Overview</h2>
          <p className="text-secondary-foreground">
            All the workflows, credentials and executions you have access to
          </p>
        </div>
        <div>insights</div>
        <div>
          <Tabs defaultValue="account">
            <TabsList>
              <TabsTrigger value="account">Account</TabsTrigger>
              <TabsTrigger value="password">Password</TabsTrigger>
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
