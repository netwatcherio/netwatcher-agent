# NetWatcher Agent

NetWater Agent is the client side of the application. It is designed to run on a host machine of the network you intend to monitor. It connects to the remote [control server](https://github.com/netwatcherio/netwatcher-control) using HTTP requests (websockets is a todo)

With Zabbix and other solutions not doing exactly what we wanted, I decided to create this...

# Installation
## Windows
1. Download the latest release or windows msi installer
   [https://github.com/netwatcherio/netwatcher-agent/releases/tag/v.1.0.2](https://github.com/netwatcherio/netwatcher-agent/releases/tag/v.1.0.2)
2. Install on host machine (eg. Windows 10 machine, or Windows 11)
3. Navigate to `C:\Program Files (x86)\NetWatcher Agent`
4. Run the agent `.exe` with Administrator permissions
   *this will generate `config.conf`*

5. Edit `config.conf` with your text editor of choice
6. Input your pin generated on the control panel in `PIN=` (eg. 123456789)
7. Input the API_URL. eg.`API_URL=https://prod.netwatcher.io`
8. Run the agent with Administrator again, and keep it open...
   **running it as a service is soon™️**
9. Check the control panel in a bit

## Planned Features
- auto update from stable releases
- iperf integration to test between set groups of "master agents"
- nmap w/ cool network layout??!?? 🤪
- iperf speedtests to master agents
- snmp

## Changelog
	Just look at commits, eventually I'll make a change log once more stable.

# License
[`GNU Affero General Public License v3.0`](https://github.com/netwatcherio/netwatcher-agent/blob/master/LICENSE.md)
