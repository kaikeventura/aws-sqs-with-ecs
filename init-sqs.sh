#!/bin/bash
awslocal sqs create-queue --queue-name teste-fila
echo "Fila 'teste-fila' criada com sucesso!"

awslocal sqs create-queue --queue-name teste-fila-2
echo "Fila 'teste-fila-2' criada com sucesso!"
