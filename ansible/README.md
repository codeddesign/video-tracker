# Video Tracker

## Deployment

This Go app is deployed using [Ansible](https://www.ansible.com/).

### Ansible Configuration

1. The hosts should be set in ```/etc/ansible/hosts``` under the ```ad3production``` group.
2. A group_var should be created in ```/etc/ansible/group_vars/ad3production``` that contains ```ansible_ssh_user: ad3```.
3. A folder for custom roles should be created: ```mkdir ~/.ansible/roles``` and set in the config file ```/etc/ansible/ansible.cfg``` under the key ```roles_path```
4. The following roles role should be installed by running:```ansible-galaxy install -r requirements.yml```.

### Install Logentries

To install the Logentries daemon, our log management system, run ```ansible-playbook logentries.yaml```.

### Install the Datadog Agent

To install the Datadog agent, our monitoring service, run ```ansible-playbook datadog.yaml```.

### Install Golang

To install Golang in the target servers, run: ```ansible-playbook install.yaml```.

Be aware that passwordless sudo must be setup in order to install Go on the target servers.

### Deploy the video-tracker

To deploy this app, run: ```ansible-playbook deploy.yaml```

> The option `--ask-vault-pass` must be used with a valid password to decrypt the `secrets.yaml` file, which contains production information.
