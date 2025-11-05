# Migration to the TeamCity Kubernetes Operator

This guide explains two practical approaches to migrate an existing TeamCity installation to the TeamCity Kubernetes Operator, using consistent terminology with this repository:
- Main Node: the primary TeamCity Server node
- Secondary TeamCity Node: an additional TeamCity Server node in a multi-node setup

For general installation and CRD usage, see the project README.


## Approach 1 — Move the TeamCity Data Directory (simplest)

This is the most straightforward and fastest approach. You move the existing TeamCity Data Directory to a PersistentVolumeClaim (PVC) used by the Operator-managed TeamCity. The database instance remains the same.

High-level steps:

1. Install the TeamCity Operator into your Kubernetes cluster.
   - See README for Helm-based installation.
2. Stop the current (non-Kubernetes) TeamCity Server.
3. Archive the existing TeamCity Data Directory.
4. Prepare the existing database connection properties and store them in a Kubernetes Secret. You will reference it via the `spec.databaseSecret.secret` field of the TeamCity custom resource (CR).
5. Create the TeamCity CR in the Kubernetes cluster.
   - Optional: set the Main Node responsibilities in advance using `spec.mainNode.spec.responsibilities: ["MAIN_NODE", "CAN_PROCESS_BUILD_MESSAGES", "CAN_CHECK_FOR_CHANGES", "CAN_PROCESS_BUILD_TRIGGERS", "CAN_PROCESS_USER_DATA_MODIFICATION_REQUESTS"]` so you don’t need to assign them manually after startup.
6. Copy the archived Data Directory into the PVC created by the Operator and referenced in `spec.dataDirVolumeClaim`.
7. Unarchive the data into the configured Data Directory path inside the container (`/storage` by default in examples).
8. Restart the Operator-managed TeamCity Server by deleting the Pod. Kubernetes will recreate it.
   - Potential improvement (future operator mode): disable the “replicas = 0” mode and then start the server.
9. First start after migration
   - If responsibilities were not changed in step 5, the Server may initially start as a Secondary TeamCity Node because the previous installation was acting as the Main Node. That’s expected if you did not restore the old main but are switching to the new Operator-managed server.
10. Promote the new Server to be the Main Node:
    - Open Administration → Nodes Configuration in the TeamCity UI and assign all Main Node responsibilities to the new server. (Skip if you already set responsibilities in step 5.)
    - Remove the previous Main Node record.

Important notes:
- Build artifacts are not copied by this procedure unless they were stored alongside the Data Directory (by default `<TeamCity Data Directory>/system/artifacts`). Plan artifact migration accordingly if they are stored externally.
- TeamCity Server logs are not copied. They are stored separately from the Data Directory.


## Approach 2 — Full Backup and Restore

Use TeamCity’s built-in backup/restore procedures. This approach requires a new, empty database for the restore.

High-level steps:

1. Install the TeamCity Operator into your Kubernetes cluster.
   - See README for Helm-based installation.
2. Create a TeamCity backup:
   - From the UI, or
   - Stop the current TeamCity Server and create a backup with the command-line tool (see TeamCity documentation).
3. Create the TeamCity CR in the Kubernetes cluster and temporarily relax probe settings to allow enough time for restore:
   - Set long `initialDelaySeconds` values for `spec.mainNode.spec.startupProbeSettings`, `readinessProbeSettings`, and `livenessProbeSettings` (several hours if unsure). You can adjust to a realistic value later.
   - It is required to set the Main Node responsibilities explicitly: `spec.mainNode.spec.responsibilities: ["MAIN_NODE", "CAN_PROCESS_BUILD_MESSAGES", "CAN_CHECK_FOR_CHANGES", "CAN_PROCESS_BUILD_TRIGGERS", "CAN_PROCESS_USER_DATA_MODIFICATION_REQUESTS"]`.
4. Upload the TeamCity backup file into the Data Directory on the PVC referenced by `spec.dataDirVolumeClaim` of the new TeamCity Server.
5. Start the restore process in the TeamCity UI:
   - Provide database connection settings for the new, empty database (follow database-specific steps in TeamCity docs).
   - Provide the backup file path in the Data Directory.
6. After the restore completes, adjust the probe settings back to normal values and restart the Pod to apply them if needed.

Important notes:
- Build artifacts are not copied by this procedure unless they were stored alongside the Data Directory (by default `<TeamCity Data Directory>/system/artifacts`). Plan artifact migration accordingly if they are stored externally.
- TeamCity Server logs are not copied. They are stored separately from the Data Directory.


## Tips and references
- Responsibilities and multi-node concepts: https://www.jetbrains.com/help/teamcity/multinode-setup.html
- TeamCity Data Directory: https://www.jetbrains.com/help/teamcity/teamcity-data-directory.html
- Manual backup and restore: https://www.jetbrains.com/help/teamcity/manual-backup-and-restore.html
- Restoring from backup: https://www.jetbrains.com/help/teamcity/restoring-teamcity-data-from-backup.html
- Database-specific steps: https://www.jetbrains.com/help/teamcity/set-up-external-database.html#Database-specific+Steps
