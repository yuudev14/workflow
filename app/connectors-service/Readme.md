for running celery workerr:

celery -A src.workers.celery worker -P gevent -c 1000 --loglevel=info

