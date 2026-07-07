# 0004 — Plain container sandbox now, behind a Sandbox interface

**Status:** Accepted

**Context:** Agent tools must run somewhere isolated. Strong isolation
(Firecracker microVMs, gVisor) is rented commodity infra, but its trigger —
untrusted input from someone other than me — isn't true yet. Industry consensus:
"your container is not a sandbox" — a plain Docker container is the weakest point
on the spectrum, fine only while single-user and trusted.

**Decision:** Use a plain Docker container as the sandbox now, behind a `Sandbox`
interface I own. Swap in a rented microVM/gVisor sandbox at the trigger, not
before.

**Alternatives:** Stand up Firecracker/gVisor immediately (rejected — pays
isolation cost with no trigger; it's rented anyway, so nothing learned by doing it
early). No isolation at all (rejected — even single-user, the interface boundary
matters).

**Consequences:** Isolation is a *rent* seam — the interim plain container is a
guilt-free stub. Swapping the walls later touches the `Sandbox` impl, not the
orchestration policy or tools. Trigger to revisit: untrusted input arrives.
