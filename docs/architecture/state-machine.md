# Sandbox state machine

~~~mermaid
stateDiagram-v2
    [*] --> requested
    requested --> provisioning
    requested --> failed
    provisioning --> starting
    provisioning --> failed
    starting --> running
    starting --> failed
    running --> snapshotting
    running --> stopping
    running --> failed
    snapshotting --> running
    snapshotting --> failed
    stopping --> stopped
    stopping --> destroyed
    stopping --> failed
    stopped --> destroyed
    failed --> destroyed
    destroyed --> [*]
~~~

The authoritative transition table is in internal/contracts/lifecycle.md. The Go control plane is the only owner of public lifecycle state. Runtime health and command events are inputs to that state machine, not alternate lifecycle state.
