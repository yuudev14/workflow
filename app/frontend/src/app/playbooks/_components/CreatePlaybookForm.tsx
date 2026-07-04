import React from 'react'

import { Button } from "@/components/ui/button";
import { zodResolver } from "@hookform/resolvers/zod"
import { useForm } from "react-hook-form"
import { z } from "zod"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { useRouter } from "next/navigation";
import { useMutation } from "@tanstack/react-query";
import PlaybookService from "@/services/playbooks/playbooks";
import { CreatePlaybookPayload } from "@/services/playbooks/playbooks.schema";
import { toast } from "@/hooks/use-toast";
import { queryClient } from "@/components/provider/main-provider";

const formSchema = z.object({
  name: z.string().min(2, {
    message: "name must be at least 2 characters.",
  }),
  description: z.string().optional(),
})


const CreatePlaybookForm = () => {
  const router = useRouter()

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      description: "",
    },
  })

  const mutation = useMutation({
    mutationFn: (workflow: CreatePlaybookPayload) => {
      return PlaybookService.createPlaybook(workflow)
    },
    onSuccess: (data) => {
      toast({
        title: "succesfully added a workflow",
        description: "redirecting you to the playground",
      })
      router.push(`/playbooks/update/${data.id}`)
      queryClient.removeQueries({
        "queryKey": ['workflow-lists']
      })
    },
    onError(error) {
      toast({
        title: "Error when adding a new workflow",
        description: error.message,
      })
    },
  })

  function onSubmit(data: z.infer<typeof formSchema>) {
    mutation.mutate(data)
  }
  return (
    <Dialog>
      <DialogTrigger className='px-4 bg-primary text-primary-foreground min-h-10'>Create Playbook</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add a new workflow</DialogTitle>
          <DialogDescription>

          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-8">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input placeholder="Automate Email.., etc." {...field} />
                  </FormControl>
                  <FormDescription>
                    Name of the workflow
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />


            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Description</FormLabel>
                  <FormControl>
                    <Textarea {...field} />
                  </FormControl>
                  <FormDescription>
                    Description of the workflow
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button type="submit" disabled={mutation.isPending} showLoader={mutation.isPending}>
              Submit</Button>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}

export default CreatePlaybookForm