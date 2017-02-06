# Video Tracker

## Deployment

This Go app is deployed using [Ansible](https://www.ansible.com/).

### Ansible Configuration

1. The hosts should be set in ```/etc/ansible/hosts``` under the ```ad3production``` group. 
2. A group_var should be created in ```/etc/ansible/group_vars/ad3production``` that contains ```ansible_ssh_user: forge```.
3. A folder for custom roles should be created: ```mkdir ~/.ansible/roles``` and set in the config file ```/etc/ansible/ansible.cfg``` under the key ```roles_path```
4. The golang role should be installed by running ```ansible-galaxy install joshualund.golang```.

### Install Golang

To install Golang in the target servers, run: ```ansible-playbook install.yaml```

### Deploy the video-tracker

To deploy this app, run: ```ansible-playbook deploy.yaml```

The required ```.env``` file is not automatically deployed, and should be manually created after deployment in ```~/video-tracker/bin/.env``` and it should contain the ```REDIS_CONNECTION``` key.
