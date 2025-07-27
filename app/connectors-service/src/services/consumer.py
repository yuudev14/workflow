import asyncio
import json
import aio_pika
import aio_pika.abc

from src.settings import settings
from src.core.workflow import WorkflowGraph


async def consume_messages(loop):
    # Connecting with the given parameters is also possible.
    # aio_pika.connect_robust(host="host", login="login", password="password")
    # You can only choose one option to create a connection, url or kw-based params.
    await asyncio.sleep(1)
    print("hi")
    connection = await aio_pika.connect_robust(
       settings.mq_url, loop=loop
    )

    async with connection:
        # Creating channel
        channel: aio_pika.abc.AbstractChannel = await connection.channel()

        # Declaring queue
        worlflow_queue: aio_pika.abc.AbstractQueue = await channel.declare_queue(
            settings.workflow_queue,
            durable=True,
            auto_delete=False,
            exclusive=False,
        )

        async with worlflow_queue.iterator() as queue_iter:
            # Cancel consuming after __aexit__
            print("listening to mq")
            async for message in queue_iter:
                async with message.process():
                    try:
                        json_body: dict = json.loads(message.body.decode())
                        print(json.dumps(json_body, indent=2))
                        graph = json_body.get("graph")
                        task_information = json_body.get("tasks")
                        workflow_history_id = json_body.get("workflow_history_id")

                        if graph is None or task_information is None or workflow_history_id is None:
                            raise Exception("either graph, task_information, or workflow_history_id is None")
                        
                        workflow = WorkflowGraph(graph=graph, task_information=task_information, workflow_history_id=workflow_history_id)
                        workflow.generate_chain_task_using_topological_sort()

                        print(json_body)

                    except Exception as e:
                        print(e)
                        
