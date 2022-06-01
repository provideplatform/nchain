#!/bin/bash

#
# Copyright 2017-2022 Provide Technologies Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

if [[ -z "${JWT_SIGNER_PUBLIC_KEY}" ]]; then
  JWT_SIGNER_PUBLIC_KEY='-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAullT/WoZnxecxKwQFlwE
9lpQrekSD+txCgtb9T3JvvX/YkZTYkerf0rssQtrwkBlDQtm2cB5mHlRt4lRDKQy
EA2qNJGM1Yu379abVObQ9ZXI2q7jTBZzL/Yl9AgUKlDIAXYFVfJ8XWVTi0l32Vsx
tJSd97hiRXO+RqQu5UEr3jJ5tL73iNLp5BitRBwa4KbDCbicWKfSH5hK5DM75EyM
R/SzR3oCLPFNLs+fyc7zH98S1atglbelkZsMk/mSIKJJl1fZFVCUxA+8CaPiKbpD
QLpzydqyrk/y275aSU/tFHidoewvtWorNyFWRnefoWOsJFlfq1crgMu2YHTMBVtU
SJ+4MS5D9fuk0queOqsVUgT7BVRSFHgDH7IpBZ8s9WRrpE6XOE+feTUyyWMjkVgn
gLm5RSbHpB8Wt/Wssy3VMPV3T5uojPvX+ITmf1utz0y41gU+iZ/YFKeNN8WysLxX
AP3Bbgo+zNLfpcrH1Y27WGBWPtHtzqiafhdfX6LQ3/zXXlNuruagjUohXaMltH+S
K8zK4j7n+BYl+7y1dzOQw4CadsDi5whgNcg2QUxuTlW+TQ5VBvdUl9wpTSygD88H
xH2b0OBcVjYsgRnQ9OZpQ+kIPaFhaWChnfEArCmhrOEgOnhfkr6YGDHFenfT3/RA
PUl1cxrvY7BHh4obNa6Bf8ECAwEAAQ==
-----END PUBLIC KEY-----'
fi

if [[ -z "${PGP_PUBLIC_KEY}" ]]; then
  PGP_PUBLIC_KEY='-----BEGIN PGP PUBLIC KEY BLOCK-----

mQINBF1Db+IBEAC0nRf3s6rls6jhWeWWTAJY8Nn4+qPUbSu0ZOx1DAqOHHxYAek1
TOuogsXaFPRtRL5mO+0aRIDjqo6GKp9IC8k6XFlJ/+LU1C09O5XOkbzhVoHtTHOY
dvLY1N3Pw5tzemFnbjMVrbTcuLgVAZoW9+e1GTUJT/VUL6AVYhg51U3r8sOuiUX5
wJrpGF4dhtOUc6pv3aBuG/iqA7vrJ8lME/3kdUZIMcs+StqJBxBuk/GykPAp5de9
vofqVd8h1aZKBjHCcdDvGDK2bLqyVk+0lE8zoh/2HG+52y/dqdVt6VEsRuf96Cou
pGeftbXKkgHv4pf0ySrNoXr3bkZmuf+SJyfF+hBq34G4zGVdT3IYH5Dwsd+ScQ72
KVI9XuO3sny+TUSWIXjWFTpQ0mhtjMhdHngXERBcgmdaS5JfmgGev3l0tqOBWhXA
oObRQ8oPhWhF20sM7LHWZSqbf4GGiVShCK6RxlRm2Uwhl1Fjx+1ThtKkq+JUgPrs
hCtk0CZVXlKIbJjrvRhJ7x/fjDEyfXur4wscHrGJr45M+3ts1dRhKUxKpNl8k9p1
RXEEntNcsV0FAZz4B0l7ImGVOKK9mdlcRLVMZ37NC5QeEoOcglHB3wGpMMmXcZU1
ZlRkQt0M/FE/PU4pKXtqiyGUZP/EFzY2O+adZZCXAbgmdbC+ktsnQCWpCwARAQAB
tCpwcm92aWRlLWRldiA8ZW5naW5lZXJpbmdAcHJvdmlkZS5zZXJ2aWNlcz6JAk4E
EwEIADgWIQS8yJWwHRuPxwaoDKhhrI+FPBEK6AUCXUNv4gIbAwULCQgHAgYVCgkI
CwIEFgIDAQIeAQIXgAAKCRBhrI+FPBEK6MHrEACm8uJ2Xc20vnXmJCuMqL3KsR75
JKcGJw8G6z3EpRjV8FeZTfOpjO/joe+X7HrCgKq8RTfnoduApEYY7Jut5Cqlw9VF
ImQMfYUBOjMzrfbkBMngjd4P7FcFAOL+amgYP85whSoKZL1EdJkpiScM16i4rvAv
LHC8BzLS8XrkF52uMV4uaFDlgI1VVhm/Q0U/9g/WJBHXXugEpnuttT3TT0rD8BLY
bIxRdsli8M0N+c2BBfISA5kNUl2j7MEqhKPuDXWHHRBkDxKwrk/mROjJWexOtUTl
GR+WFPs2W4ikhhX51zPUCYnrhm/WFrjy+xwNveaNGrk4pr3Hm6YEGnTD5H71t4vW
ezlpbfA7aPLq6/HX30JsLUpFyl/PTE4BAAizuu04HbnRHlNBsITa7pbf2MMN4tSe
3uxcHql8BDmm3RSSK2N1vwMszySPmapUVKi3W6qduTBgT9WCd77mt20259Lyurqu
s7UF0CvrHfxGeyrMkXbv13xwG+wqWpI2hmiOTkEg2zhxap16sWhQVNJP/r1sqqUa
BwQ8cGI62hEwd9rF49etznl4TmzqUvQCx+VEnbArT/7nUKHmGPXAcI7g0hMJj0N6
LQXF5w8xlHFXOfzpDNN7YzobARgiWC41NezPZKhv4y8Cy2riUsju5qRyIN37Ipy1
rE2pI5SJ+jU80Z+G2bkCDQRdQ2/iARAArsi1Zuqv9raZWLLcAgyk1ed6KtmvWxTH
/zf6E37P2dh+Z/7pGJp6xkHUdtgaj7EABEZfzGJnyo11Pna7u8se0j1ZtxwazjS+
B48KW9eM0K9m8eje0RkP99jg3naZGV1LnJKt2g68h5rsibfBJYg7/tYxBpqILavW
CCg9xTDsN0X9D5hQdg2pclG9OsSmK72rBbzuEkKnhY99U+2LluCIi5bX07gYuZ9G
lbWtX6IKlvPx/FDLaOqUWrtJ0WYQMb92TqimbRe6MG2vLzPIKrn8MkaSSADd9nhT
rnKD61hr4kFr+sd+iyr1nKg97ppd4dWaL88oDEeKE9xO3H391ihz0stbt2AfD9R0
2i86UfY06lLfEukiqlwUGnvxVntjDZmqJWZBG7mLtilSC1THpifPMJ2OogSJRMJ9
wZVelI2aK6uSrqkzW8dutKhcRmlRVDCtMdpFJncxuyLneVWKBMH8SMVenYrbrkqq
1NN/25H4Wy7A/4MCvMHoBRMQheiFBEWx+JEkWnDFeVA2ZGREmQsqncG81GN/pULW
shlCEgMU3LrKoB2wzfgBjVXjKq/+7utuwKshkEH2y6g5IPDOOHtfdXNqZ3Ph3r0l
k/KqWY56csJ0ytUXVYpcr1u7VxS/yoOQQBkh481CSJfV7eRJ5/epEQKdxP4sPFCv
00SZv1/XyvkAEQEAAYkCNgQYAQgAIBYhBLzIlbAdG4/HBqgMqGGsj4U8EQroBQJd
Q2/iAhsMAAoJEGGsj4U8EQroTusP/jKLeLOaqXWIlZqSWR5kaYUWeEvtZWVWQLsa
kqKEIFwTSq5HBWIfyZxtPTmo3JN+78ufkYJXg0Hs/sTIWbs3juoDSgLvql2ME1M2
a3Dgr2OPkMPnEReQ8jlcRcseJqqdl2CpFG6pQzmof8UZZ3G//GHIop6Fss29+hjg
fDAOc/YXwcrqVs0cPpXSBU+XJwH7zIvkVfRKFeoieOqK8mHhgDP6FTJ2tyR8+UDa
NsaB7rF2LJyh4UtJLImlftH/1yYWjaYxjsOqiqiTA5NPbiEKtfnK54740YVJouAq
CCoQV6vwMbvre3izUSCPcmHWdo1Lpy8MV1vrDto7biptubqM6Sli0BUlP6/EhFnY
DaKMm9iZvsS6q5mHIcmZxuqMkJ6gheMISqN4tzoGshOpE+XPRk6GgJp6GxTsEuim
MBjvqsdVq8lqp1kGWdSZDgnOPOPwvW3mIp8nEwY7b8hp/YGz6pylaQaCKeACQkfd
r82Hnp5/o36qMdudanGGvP28qbXmEan/VyKQuReBJ2JQKLUpFpCWUOaho6dcP6Zo
jNfRxN+DUEJNER0oZUTEeEno3BfRYkpQ/EZjtQ9muVh2S8UVL06OV0f5deOxicP4
65KorIgQeczCg8iX3Pt1ZojYNW4YOnrEys22ZaI6+hmvLf9Zx4u4ip+tTIaEkJoi
96kKxfZ6
=HwI4
-----END PGP PUBLIC KEY BLOCK-----'
fi

if [[ -z "${PGP_PRIVATE_KEY}" ]]; then
  PGP_PRIVATE_KEY='-----BEGIN PGP PRIVATE KEY BLOCK-----

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
=1A2t
-----END PGP PRIVATE KEY BLOCK-----'
fi

if [[ -z "${PGP_PASSPHRASE}" ]]; then
  PGP_PASSPHRASE=password
fi

if [[ -z "${LOG_LEVEL}" ]]; then
  LOG_LEVEL=info
fi

if [[ -z "${DATABASE_HOST}" ]]; then
  DATABASE_HOST=localhost
fi

if [[ -z "${DATABASE_NAME}" ]]; then
  DATABASE_NAME=nchain_dev
fi

if [[ -z "${DATABASE_USER}" ]]; then
  DATABASE_USER=nchain
fi

if [[ -z "${DATABASE_PASSWORD}" ]]; then
  DATABASE_PASSWORD=
fi

if [[ -z "${DATABASE_LOGGING}" ]]; then
  DATABASE_LOGGING=false
fi

if [[ -z "${CONSUME_NATS_STREAMING_SUBSCRIPTIONS}" ]]; then
  CONSUME_NATS_STREAMING_SUBSCRIPTIONS=true
fi

if [[ -z "${NATS_CLUSTER_ID}" ]]; then
  NATS_CLUSTER_ID=provide
fi

if [[ -z "${NATS_TOKEN}" ]]; then
  NATS_TOKEN=testtoken
fi

if [[ -z "${NATS_URL}" ]]; then
  NATS_URL=nats://localhost:4222
fi

if [[ -z "${NATS_JETSTREAM_URL}" ]]; then
  NATS_JETSTREAM_URL=nats://localhost:4222
fi

if [[ -z "${NATS_STREAMING_CONCURRENCY}" ]]; then
  NATS_STREAMING_CONCURRENCY=1
fi

if [[ -z "${NATS_FORCE_TLS}" ]]; then
  NATS_FORCE_TLS=false
fi

#NATS_ROOT_CA_CERTIFICATES=/Users/kt/selfsigned-ca/ca.pem \
#NATS_TLS_CERTIFICATES='{"/Users/kt/selfsigned-ca/peer.key": "/Users/kt/selfsigned-ca/peer.crt"}' \

if [[ -z "${REDIS_HOSTS}" ]]; then
  REDIS_HOSTS=localhost:6379
fi

if [[ -z "${REDIS_DB_INDEX}" ]]; then
  REDIS_DB_INDEX=1
fi

DATABASE_HOST=$DATABASE_HOST DATABASE_USER=$DATABASE_USER DATABASE_PASSWORD=$DATABASE_PASSWORD DATABASE_NAME=$DATABASE_NAME ./ops/migrate.sh

JWT_SIGNER_PUBLIC_KEY=$JWT_SIGNER_PUBLIC_KEY \
PGP_PUBLIC_KEY=$PGP_PUBLIC_KEY \
PGP_PRIVATE_KEY=$PGP_PRIVATE_KEY \
PGP_PASSPHRASE=$PGP_PASSPHRASE \
CONSUME_NATS_STREAMING_SUBSCRIPTIONS=$CONSUME_NATS_STREAMING_SUBSCRIPTIONS \
NATS_CLUSTER_ID=$NATS_CLUSTER_ID \
NATS_TOKEN=$NATS_TOKEN \
NATS_URL=$NATS_URL \
NATS_JETSTREAM_URL=$NATS_JETSTREAM_URL \
NATS_STREAMING_CONCURRENCY=$NATS_STREAMING_CONCURRENCY \
NATS_FORCE_TLS=$NATS_FORCE_TLS \
DATABASE_HOST=$DATABASE_HOST \
DATABASE_NAME=$DATABASE_NAME \
DATABASE_LOGGING=$DATABSE_LOGGING \
DATABASE_USER=$DATABASE_USER \
DATABASE_PASSWORD=$DATABASE_PASSWORD \
LOG_LEVEL=$LOG_LEVEL \
REDIS_HOSTS=$REDIS_HOSTS \
REDIS_DB_INDEX=$REDIS_DB_INDEX \
SYSLOG_ENDPOINT=${SYSLOG_ENDPOINT} \
./.bin/nchain_consumer
