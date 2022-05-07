import boto3
import config_pb2
from audioop import add
from urllib import response
from venv import create
from google.protobuf import text_format as _text_format

regions = ["us-east-1","us-west-1","ap-northeast-1","ap-southeast-2","ap-southeast-1","ap-east-1","eu-central-1","eu-west-1","eu-west-3","me-south-1","sa-east-1","ca-central-1"]
access_key = ""
secret_key = ""

instances = []
for region in regions:
  ec2 = boto3.client('ec2', aws_access_key_id=access_key, aws_secret_access_key=secret_key, region_name=region)
  Tags = [{'Key': 'Name', 'Value': 'Free'}]
  Filter = [
    {
      'Name': 'key-name',
      'Values': [
        'dumbo',
      ]
    }
  ]
  response = ec2.describe_instances(Filters=Filter)
  for i in range(len(response['Reservations'])):
    instances.append(response['Reservations'][i]['Instances'])

print(len(instances))

addrs = []
for i in range(len(instances)):
  addrs.append(instances[i]['PublicIpAddress'])

print(len(addrs))

peers = []
for i in range(len(addrs)):
  peer = config_pb2.Peer()
  peer.id = i
  peer.addr = addrs[i]+':6100'
  peer.client = ':6000'
  peers.append(peer)

CK = '.'
PK = '.'
BLS = '.'
for i in range(3):
  config = config_pb2.Configuration()
  config.id = i
  config.batch = 1
  config.n = 3
  config.f = 1
  config.delta = 500
  config.deltaBar = 2500
  config.sign = False
  config.blsKeyPath = BLS
  config.ck = CK
  config.pk = PK
  for peer in peers:
    config.peers.append(peer)
  text = _text_format.MessageToString(config)
  file = open('node'+str(i)+'.config', 'w+')
  file.write(text)
  file.close()
