#!/bin/bash
awslocal sqs create-queue --queue-name teste-fila
echo "Fila 'teste-fila' criada com sucesso!"