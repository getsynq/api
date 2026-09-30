"""
Users Management Example - list workspace members and manage their roles

Commands:
    python users.py list
    python users.py add-role <identity> <role>
    python users.py remove-role <identity> <role>
    python users.py set-roles <identity> <role> [<role> ...]

<identity> is an email address or any identity string from `list`
("synq:<user_id>", "email:<email>", "slack:<id>", "msteams:<id>").
<role> is one of: admin, developer, analyst, business_user.

Prerequisites:
- SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET environment variables
  (Read Users scope for `list`, Edit Users scope for the role commands)
- Optional API_ENDPOINT (defaults to developer.synq.io)
"""

import argparse
import os

import grpc
from dotenv import load_dotenv

from auth import TokenAuth, TokenSource
from synq.users.v1 import users_pb2, users_service_pb2, users_service_pb2_grpc

load_dotenv()

CLIENT_ID = os.getenv("SYNQ_CLIENT_ID")
CLIENT_SECRET = os.getenv("SYNQ_CLIENT_SECRET")
API_ENDPOINT = os.getenv("API_ENDPOINT", "developer.synq.io")

if CLIENT_ID is None or CLIENT_SECRET is None:
    raise Exception("SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET must be set")

ROLES = {
    "admin": users_pb2.USER_ROLE_ADMIN,
    "developer": users_pb2.USER_ROLE_DEVELOPER,
    "analyst": users_pb2.USER_ROLE_ANALYST,
    "business_user": users_pb2.USER_ROLE_BUSINESS_USER,
}
ROLE_NAMES = {value: name for name, value in ROLES.items()}

STATUS_NAMES = {
    users_pb2.USER_STATUS_ACTIVE: "active",
    users_pb2.USER_STATUS_INVITED: "invited",
}


def to_identity(value: str) -> str:
    """Accepts a bare email address as shorthand for "email:<address>"."""
    if ":" not in value and "@" in value:
        return f"email:{value}"
    return value


def format_user(user: users_pb2.User) -> str:
    name = f"{user.first_name} {user.last_name}".strip() or "-"
    roles = ",".join(ROLE_NAMES.get(r, str(r)) for r in user.roles) or "-"
    status = STATUS_NAMES.get(user.status, "-")
    last_login = user.last_login_at.ToDatetime().isoformat() + "Z" if user.HasField("last_login_at") else "never"
    return f"{user.email:<40} {name:<30} {roles:<30} {status:<8} {last_login}"


def list_users(stub):
    resp = stub.ListUsers(users_service_pb2.ListUsersRequest())
    print(f"{'EMAIL':<40} {'NAME':<30} {'ROLES':<30} {'STATUS':<8} LAST LOGIN")
    for user in sorted(resp.users, key=lambda u: u.email):
        print(format_user(user))
    print(f"\n{len(resp.users)} users")


def find_user(stub, identity: str):
    """Returns the workspace member holding the identity, or None when there is none."""
    email = identity.removeprefix("email:").lower() if identity.startswith("email:") else None
    for user in stub.ListUsers(users_service_pb2.ListUsersRequest()).users:
        if identity in user.identities or (email and user.email.lower() == email):
            return user
    return None


def add_role(stub, identity: str, role: str):
    resp = stub.AddUserRole(users_service_pb2.AddUserRoleRequest(identity=identity, role=ROLES[role]))
    print(f"Added {role} to {identity}")
    print(format_user(resp.user))


def remove_role(stub, identity: str, role: str):
    resp = stub.RemoveUserRole(users_service_pb2.RemoveUserRoleRequest(identity=identity, role=ROLES[role]))
    print(f"Removed {role} from {identity}")
    if resp.HasField("user"):
        print(format_user(resp.user))
    else:
        print("User is not a member of the workspace")


def set_roles(stub, identity: str, roles: list):
    """Makes the user hold exactly the given roles, adding before removing so they never lose membership midway."""
    user = find_user(stub, identity)
    current = {ROLE_NAMES[r] for r in user.roles if r in ROLE_NAMES} if user else set()
    wanted = set(roles)

    for role in sorted(wanted - current):
        add_role(stub, identity, role)
    for role in sorted(current - wanted):
        remove_role(stub, identity, role)
    if current == wanted:
        print(f"{identity} already holds {', '.join(sorted(wanted))}")


def main():
    parser = argparse.ArgumentParser(description="List workspace members and manage their roles.")
    commands = parser.add_subparsers(dest="command", required=True)

    commands.add_parser("list", help="List all members with their roles, status and last sign-in")

    for name, help_text in (("add-role", "Grant a role"), ("remove-role", "Withdraw a role")):
        cmd = commands.add_parser(name, help=help_text)
        cmd.add_argument("identity")
        cmd.add_argument("role", choices=ROLES)

    cmd = commands.add_parser("set-roles", help="Make the user hold exactly these roles")
    cmd.add_argument("identity")
    cmd.add_argument("roles", nargs="+", choices=ROLES)

    args = parser.parse_args()

    token_source = TokenSource(CLIENT_ID, CLIENT_SECRET, API_ENDPOINT)
    grpc_credentials = grpc.metadata_call_credentials(TokenAuth(token_source))

    with grpc.secure_channel(
        f"{API_ENDPOINT}:443",
        grpc.composite_channel_credentials(grpc.ssl_channel_credentials(), grpc_credentials),
        options=(("grpc.default_authority", API_ENDPOINT),),
    ) as channel:
        grpc.channel_ready_future(channel).result(timeout=10)
        stub = users_service_pb2_grpc.UsersServiceStub(channel)

        if args.command == "list":
            list_users(stub)
        elif args.command == "add-role":
            add_role(stub, to_identity(args.identity), args.role)
        elif args.command == "remove-role":
            remove_role(stub, to_identity(args.identity), args.role)
        elif args.command == "set-roles":
            set_roles(stub, to_identity(args.identity), args.roles)


if __name__ == "__main__":
    main()
