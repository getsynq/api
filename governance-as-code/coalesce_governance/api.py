"""
Thin API client + mapping between the YAML governance model and the public
protos, in both directions (apply: dict -> proto; export/pull: proto -> dict).
"""

from synq.dataproducts.v2 import (
    dataproducts_service_pb2 as dp_svc,
    dataproducts_service_pb2_grpc as dp_grpc,
    dataproduct_pb2 as dp,
    dataproduct_definition_pb2 as dp_def,
)
from synq.owners.v1 import (
    owners_service_pb2 as ow_svc,
    owners_service_pb2_grpc as ow_grpc,
    ownership_pb2 as own,
    contact_pb2 as contact,
)
from synq.alerts.v1 import alerts_pb2
from synq.v1 import severity_pb2, pagination_pb2

_PAGE = 500


# ---- scalar enum helpers ---------------------------------------------------

def priority_to_proto(name):
    return dp.Dataproduct.Priority.Value(f"PRIORITY_{name.upper()}")


def priority_from_proto(value):
    # PRIORITY_P1 -> "P1"; UNSPECIFIED -> None
    name = dp.Dataproduct.Priority.Name(value)
    return name[len("PRIORITY_"):] if name.startswith("PRIORITY_P") else None


def severity_to_proto(name):
    return severity_pb2.Severity.Value(f"SEVERITY_{name.upper()}")


def severity_from_proto(value):
    return severity_pb2.Severity.Name(value)[len("SEVERITY_"):]


def ongoing_to_proto(spec):
    if spec in (None, "disabled"):
        return alerts_pb2.OngoingAlertsStrategy(disabled=alerts_pb2.OngoingAlertsStrategy.Disabled())
    if spec == "stream":
        return alerts_pb2.OngoingAlertsStrategy(stream=alerts_pb2.OngoingAlertsStrategy.Stream())
    if isinstance(spec, dict) and "schedule" in spec:
        return alerts_pb2.OngoingAlertsStrategy(
            schedule=alerts_pb2.OngoingAlertsStrategy.Schedule(cron=spec["schedule"]))
    raise ValueError(f"invalid ongoing spec: {spec!r}")


def ongoing_from_proto(msg):
    which = msg.WhichOneof("strategy")
    if which == "schedule":
        return {"schedule": msg.schedule.cron}
    if which == "stream":
        return "stream"
    return "disabled"


# ---- contacts --------------------------------------------------------------

def contact_to_proto(entry):
    if "slack" in entry:
        return contact.Contact(slack=contact.SlackChannelContact(channel=entry["slack"]))
    if "email" in entry:
        return contact.Contact(email=contact.EmailContact(recipient_emails=list(entry["email"])))
    if "users" in entry:
        return contact.Contact(users=contact.UserContact(user_emails=list(entry["users"])))
    if "ms_teams" in entry:
        return contact.Contact(ms_teams=contact.MsTeamsContact(channel_id=entry["ms_teams"]))
    raise ValueError(f"invalid contact entry: {entry!r}")


def contact_from_proto(c):
    which = c.WhichOneof("contact_method")
    if which == "slack":
        return {"slack": c.slack.channel}
    if which == "email":
        return {"email": list(c.email.recipient_emails)}
    if which == "users":
        return {"users": list(c.users.user_emails)}
    if which == "ms_teams":
        return {"ms_teams": c.ms_teams.channel_id}
    return {}


# ---- alert config ----------------------------------------------------------

def alert_to_proto(spec):
    spec = spec or {}
    return own.AlertConfig(
        severities=[severity_to_proto(s) for s in spec.get("severities", [])],
        notify_upstream=spec.get("notify_upstream", False),
        allow_sql_test_audit_link=spec.get("allow_sql_test_audit_link", False),
        ongoing=ongoing_to_proto(spec.get("ongoing")),
    )


def alert_from_proto(a):
    out = {}
    if a.severities:
        out["severities"] = [severity_from_proto(s) for s in a.severities]
    if a.notify_upstream:
        out["notify_upstream"] = True
    if a.allow_sql_test_audit_link:
        out["allow_sql_test_audit_link"] = True
    out["ongoing"] = ongoing_from_proto(a.ongoing)
    return out


# ---- definition ------------------------------------------------------------

def definition_to_proto(resolver_ql, part_id):
    return dp_def.DataproductDefinition(parts=[
        dp_def.DataproductDefinition.Part(
            id=part_id, query=dp_def.DataproductQuery(resolver_ql=resolver_ql))
    ])


def rendered_resolver_ql(product):
    """Best-effort single-string ResolverQL for a product's definition (rendered).
    One query part -> its rendered form; multiple -> or(...) of the parts."""
    qs, statics = [], []
    for p in product.definition.parts:
        if p.HasField("query") and p.query.rendered_resolver_ql:
            qs.append(p.query.rendered_resolver_ql)
        elif p.entity_id:
            statics.append(p.entity_id)
    if statics:
        qs.append('static_paths([{}])'.format(", ".join(f'"{s}"' for s in statics)))
    if not qs:
        return ""
    return qs[0] if len(qs) == 1 else "or({})".format(", ".join(qs))


# ---- client ----------------------------------------------------------------

class Client:
    def __init__(self, channel):
        self.dp = dp_grpc.DataproductsServiceStub(channel)
        self.ow = ow_grpc.OwnersServiceStub(channel)

    # products
    def get_products(self, ids):
        """Return {id: Dataproduct} for the given ids (missing ids omitted)."""
        if not ids:
            return {}
        return dict(self.dp.BatchGet(dp_svc.BatchGetRequest(ids=list(ids))).dataproducts)

    def list_products(self):
        out, cursor = [], None
        while True:
            pg = pagination_pb2.Pagination(page_size=_PAGE)
            if cursor:
                pg.cursor = cursor
            resp = self.dp.List(dp_svc.ListRequest(pagination=pg))
            out.extend(resp.dataproducts)
            if not resp.page_info.last_id or not resp.dataproducts:
                break
            cursor = resp.page_info.last_id
        return out

    def upsert_product(self, pid, title=None, folder=None, priority=None, resolver_ql=None, part_id=None):
        req = dp_svc.UpsertRequest(id=pid)
        if title is not None:
            req.title = title
        if folder is not None:
            req.folder = folder
        if priority is not None:
            req.priority = priority_to_proto(priority)
        if resolver_ql is not None:
            req.definition.CopyFrom(definition_to_proto(resolver_ql, part_id))
        return self.dp.Upsert(req).dataproduct

    def delete_product(self, pid):
        self.dp.Delete(dp_svc.DeleteRequest(id=pid, purge=True))

    def count_members(self, pid):
        n, cursor = 0, None
        while True:
            pg = pagination_pb2.Pagination(page_size=1000)
            if cursor:
                pg.cursor = cursor
            resp = self.dp.ListMembers(dp_svc.ListMembersRequest(id=pid, pagination=pg))
            n += len(resp.entity_ids)
            if not resp.page_info.last_id or not resp.entity_ids:
                break
            cursor = resp.page_info.last_id
        return n

    # owners
    def get_owners(self, ids):
        if not ids:
            return {}
        return dict(self.ow.BatchGetOwners(ow_svc.BatchGetOwnersRequest(ids=list(ids))).owners)

    def list_owners(self):
        out, cursor = [], None
        while True:
            pg = pagination_pb2.Pagination(page_size=_PAGE)
            if cursor:
                pg.cursor = cursor
            resp = self.ow.ListOwners(ow_svc.ListOwnersRequest(pagination=pg))
            out.extend(resp.owners)
            if not resp.page_info.last_id or not resp.owners:
                break
            cursor = resp.page_info.last_id
        return out

    def upsert_owner(self, oid, title=None, contacts=None):
        req = ow_svc.UpsertOwnerRequest(id=oid)
        if title is not None:
            req.title = title
        if contacts is not None:
            req.contacts.CopyFrom(ow_svc.ContactList(contacts=[contact_to_proto(c) for c in contacts]))
        return self.ow.UpsertOwner(req).owner

    def delete_owner(self, oid):
        self.ow.DeleteOwner(ow_svc.DeleteOwnerRequest(id=oid, purge=True))

    def list_ownerships(self, owner_id):
        out, cursor = [], None
        while True:
            pg = pagination_pb2.Pagination(page_size=_PAGE)
            if cursor:
                pg.cursor = cursor
            resp = self.ow.ListOwnerships(ow_svc.ListOwnershipsRequest(owner_id=owner_id, pagination=pg))
            out.extend(resp.ownerships)
            if not resp.page_info.last_id or not resp.ownerships:
                break
            cursor = resp.page_info.last_id
        return out

    def upsert_ownership(self, owner_id, oid, selection, alert):
        return self.ow.UpsertOwnership(ow_svc.UpsertOwnershipRequest(
            owner_id=owner_id, id=oid, selection=selection, alert=alert)).ownership

    def delete_ownership(self, oid):
        self.ow.DeleteOwnership(ow_svc.DeleteOwnershipRequest(id=oid))


def selection_to_proto(spec, resolve_ref):
    """spec is an ownership dict; resolve_ref maps a product key/id -> id."""
    if "dataproduct" in spec:
        return own.OwnershipSelection(dataproduct_id=resolve_ref(spec["dataproduct"]))
    q = spec["query"]
    return own.OwnershipSelection(
        query=own.OwnershipQuery(name=q.get("name", ""), resolver_ql=q["resolver_ql"]))


def selection_from_proto(sel):
    if sel.HasField("dataproduct_id"):
        return {"dataproduct_id": sel.dataproduct_id}
    return {"query": {"name": sel.query.name, "resolver_ql": sel.query.rendered_resolver_ql}}
