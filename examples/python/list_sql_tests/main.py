import os
import grpc

from dotenv import load_dotenv

load_dotenv()

from auth import TokenSource, TokenAuth
from synq.datachecks.sqltests.v1 import (
    sql_tests_service_pb2,
    sql_tests_service_pb2_grpc,
)

# Set the client ID and secret in the environment or in a .env file.
# The credentials need the Read SQL Tests scope.
CLIENT_ID = os.getenv("QUALITY_CLIENT_ID") or os.getenv("SYNQ_CLIENT_ID")
CLIENT_SECRET = os.getenv("QUALITY_CLIENT_SECRET") or os.getenv("SYNQ_CLIENT_SECRET")
API_ENDPOINT = os.getenv("QUALITY_API_ENDPOINT") or os.getenv("API_ENDPOINT") or "developer.synq.io"

if CLIENT_ID is None or CLIENT_SECRET is None:
    raise Exception("missing QUALITY_CLIENT_ID or QUALITY_CLIENT_SECRET env vars")

# initialize authorization
token_source = TokenSource(CLIENT_ID, CLIENT_SECRET, API_ENDPOINT)
auth_plugin = TokenAuth(token_source)
grpc_credentials = grpc.metadata_call_credentials(auth_plugin)

# create and use channel to make requests
with grpc.secure_channel(
    f"{API_ENDPOINT}:443",
    grpc.composite_channel_credentials(
        grpc.ssl_channel_credentials(),
        grpc_credentials,
    ),
    options=(("grpc.default_authority", API_ENDPOINT),),
) as channel:
    grpc.channel_ready_future(channel).result(timeout=10)
    print("grpc channel ready")

    try:
        stub = sql_tests_service_pb2_grpc.SqlTestsServiceStub(channel)
        response = stub.ListSqlTests(sql_tests_service_pb2.ListSqlTestsRequest())
        print(response)
    except grpc.RpcError as e:
        print("error listing sql tests")
        raise e
