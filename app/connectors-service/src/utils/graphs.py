# import asyncio
# import json
# import aio_pika
# import aio_pika.abc

# from src import constants


# async def main(loop):
#     # Connecting with the given parameters is also possible.
#     # aio_pika.connect_robust(host="host", login="login", password="password")
#     # You can only choose one option to create a connection, url or kw-based params.
#     connection = await aio_pika.connect_robust(
#        constants.MQ_URL, loop=loop
#     )

#     async with connection:
#         # Creating channel
#         channel: aio_pika.abc.AbstractChannel = await connection.channel()

#         # Declaring queue
#         worlflow_queue: aio_pika.abc.AbstractQueue = await channel.declare_queue(
#             constants.WORKFLOW_QUEUE,
#             durable=True,
#             auto_delete=False,
#             exclusive=False,
#         )

#         workflow_processor_queue: aio_pika.abc.AbstractQueue = await channel.declare_queue(
#             constants.WORKFLOW_PROCESSOR_QUEUE,
#             durable=True,
#             auto_delete=False,
#             exclusive=False,
#         )

#         async with worlflow_queue.iterator() as queue_iter:
#             # Cancel consuming after __aexit__
#             async for message in queue_iter:
#                 async with message.process():
#                     json_body: dict = json.loads(message.body.decode())
#                     print(json_body.get("current_node", None))
#                     await channel.default_exchange.publish(
#                         aio_pika.Message(body=message.body),
#                         routing_key=workflow_processor_queue.name,
#                     )


# if __name__ == "__main__":
#     loop = asyncio.get_event_loop()
#     loop.run_until_complete(main(loop))
#     loop.close()