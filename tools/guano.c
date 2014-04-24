/* Copyright 2014 Ooyala, Inc. All rights reserved.
 *
 * This file is licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is
 * distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

#define _GNU_SOURCE
#include <stdio.h>
#include <string.h>

#include <sched.h>

#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>

#include <sys/socket.h>
#include <net/if.h>
#include <netinet/in.h>

#include <stdlib.h>

#include <linux/ethtool.h>
#include <linux/sockios.h>
#include <linux/types.h>

int set_network_ns(char *pid)
{
	char ns[256];

	if (sprintf(ns, "/proc/%s/ns/net", pid) < 0)
		return -1;

	int fd = open(ns, O_RDWR);
	if (fd < 0)
		return -1;

	if (setns(fd, CLONE_NEWNET) < 0) {
		close(fd);
		return -1;
	}

	return 0;
}

int num_stats(int fd, struct ifreq *ifr)
{
	struct ethtool_drvinfo drvinfo;
	drvinfo.cmd = ETHTOOL_GDRVINFO;

	ifr->ifr_data = (caddr_t)&drvinfo;
	if (ioctl(fd, SIOCETHTOOL, ifr) < 0)
		return -1;

	return drvinfo.n_stats;
}

int stat_index(int fd, struct ifreq *ifr, int n, char *stat)
{
	size_t len = sizeof(struct ethtool_gstrings) + n * ETH_GSTRING_LEN;

	struct ethtool_gstrings *strings = calloc(1, len);
	if (!strings)
		return -1;

	strings->cmd = ETHTOOL_GSTRINGS;
	strings->string_set = ETH_SS_STATS;
	strings->len = n;

	ifr->ifr_data = (caddr_t)strings;
	if (ioctl(fd, SIOCETHTOOL, ifr) < 0) {
		free(strings);
		return -1;
	}

	int i = 0;
	while (i < n) {
		if (!strncmp(stat, &strings->data[i * ETH_GSTRING_LEN],  ETH_GSTRING_LEN))
			break;
		++i;
	}

	free(strings);

	return i < n ? i : -1;
}

int stat_data(int fd, struct ifreq *ifr, int n, int i)
{
	size_t len = sizeof(struct ethtool_stats) + n * sizeof(__u64);

	struct ethtool_stats *stats = calloc(1, len);
	if (!stats)
		return -1;

	stats->cmd = ETHTOOL_GSTATS;
	stats->n_stats = n;

	ifr->ifr_data = (caddr_t)stats;
	if (ioctl(fd, SIOCETHTOOL, ifr) < 0)
		return -1;

	return (int)stats->data[i];
}

int peer_ifindex(char* ifname)
{
	int ifindex = -1;

	int fd = socket(PF_INET, SOCK_DGRAM, IPPROTO_IP);
	if (fd < 0)
		return -1;

	struct ifreq ifr;
	memset(&ifr, 0, sizeof(ifr));
	strcpy(ifr.ifr_name, "eth0");

	int n = num_stats(fd, &ifr);
	if (n < 0)
		goto error;

	int i = stat_index(fd, &ifr, n, "peer_ifindex");
	if (i < 0)
		goto error;

	ifindex = stat_data(fd, &ifr, n, i);

error:
	close(fd);
	return ifindex;
}

int main(int argc, char **argv)
{
	int fd[2], ifindex;
	pid_t child;

	pipe(fd);

	child = fork();
	if (child < 0) {
		fprintf(stderr, "error: cannot fork\n");
		exit(1);
	}

	if (child == 0) {
		close(fd[0]);

		if (set_network_ns(argv[1]) < 0) {
			fprintf(stderr, "error: cannot set network namespace\n");
			exit(1);
		}

		ifindex = peer_ifindex(argv[2]);
		if (ifindex < 0) {
			fprintf(stderr, "error: cannot find peer ifindex\n");
			exit(1);
		}
		write(fd[1], &ifindex, sizeof(ifindex));

		exit(0);
	} else {
		close(fd[1]);

		if (read(fd[0], &ifindex, sizeof(ifindex)) <= 0) {
			fprintf(stderr, "error: cannot read from child\n");
			exit(1);
		}

		char ifname[IF_NAMESIZE];
		if (if_indextoname(ifindex, ifname) < 0) {
			fprintf(stderr, "error: cannot find name of iface %d\n", ifindex);
			exit(1);
		}

		printf("%d %s", ifindex, ifname);

		exit(0);
	}
}