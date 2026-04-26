# Hiatus, Unsupported Versions, Rate Limiting & Self Hosting

The creator of this bot is on an extended break, with no ETA to return to this project.

The below instructions are kept as a guide for existing installs. The share community version of the bot is rate limited. It sometimes works and sometimes doesn't.

**Using the shared community version of this bot is no longer supported and not recommended.**

Self-Hosting the bot (via Azure, AWS, Oracle Cloud, Docker instances) is provided via the Discord as a best effort process, but the underlying code is no longer being actively maintained. This message will be removed when this is no longer the case

-- 15-JAN-2023

## Modernization Changelog

The bot owner's last upstream commit in this checkout is `8cd5bdc` from
2023-01-15, titled `Added #Hiatus Information`. The changes below document the
local modernization work completed after that upstream commit. The main goal was
to make the project easier to self-host in Docker without requiring Go or extra
build tools on the host machine.

### 2026-04-27

- Replaced the old Dockerfile with a multi-stage Docker build.
  - It builds the bot from this checkout using `golang:1.26.2-bookworm`.
  - It copies only the compiled binary into a `debian:bookworm-slim` image.
  - It runs the bot as a non-root `autodelete` user.
  - It no longer rebuilds the app every time the container starts.
  - It no longer uses `go get`, `apt upgrade`, or insecure apt flags.
- Added `.dockerignore` so local config, data, logs, docs, and git history
  do not get copied into the Docker build.
- Added `compose.yml` so the bot can run with `docker compose up --build -d`.
  The compose file uses a named Docker volume for `/autodelete/data`.
- Updated the Docker instructions in this README so the project can be built
  in Docker without installing Go on the host.
- Updated `config.example.yml` so the default listener works inside Docker
  with published ports.
- Changed the module declaration from `go 1.13` to `go 1.22`. The Docker
  builder currently uses Go `1.26.2`.
- Replaced old `io/ioutil` calls with `os.ReadFile` and `os.ReadDir`.
- Removed `github.com/pkg/errors` and used standard Go error wrapping instead.
- Made channel config saves safer by writing to a temporary file first, then
  renaming it into place after the write succeeds.
- Tightened startup checks:
  - The bot now stops if the token is missing.
  - Non-sharded runs default to shard `0`.
  - Sharded runs reject invalid shard IDs.
- Stopped the OAuth callback from printing token objects to the logs.
- Kept the old Discord OAuth `invalid_client` workaround, but only when
  `clientsecret` is intentionally empty.
- Fixed queue bookkeeping so disabled channels are removed from in-flight work
  tracking.
- Changed bad Discord message timestamps from a crash into a warning and skip.
- Replaced the old `go get` setup step in `docs/legacy/setup.sh` with an explicit
  `git clone`.
- Added `/autodelete` slash command support while keeping all existing
  mention-based commands.
  - `/autodelete help`
  - `/autodelete check`
  - `/autodelete set duration:<duration> count:<count>`
- Updated the OAuth invite flow to include the `applications.commands` scope
  needed for slash commands.
- Added `slash_commands: true` to the example config. Set it to `false` to skip
  global slash-command registration.
- Added `/healthz` to the public and private HTTP listeners.
- Added a Docker `HEALTHCHECK` using a small compiled healthcheck binary.
- Added a GitHub Actions workflow that validates Compose, builds the Docker
  build stage, runs `go test` inside Docker, and builds the runtime image.
- Moved the original operator-specific deployment files into `docs/legacy/`.
  Docker and Compose are now the supported self-hosting path for this checkout.
- Added focused unit tests for slash-command argument parsing and the `/healthz`
  HTTP handler.

Validation completed:

- `docker compose config`
- `git diff --check`
- `bash -n docs/legacy/setup.sh`
- `bash -n docs/legacy/build.sh`

Validation not completed:

- Docker image build was attempted but blocked because Docker socket access was
  unavailable.
- Go tests were not run locally because Go was intentionally not installed on
  the host. The GitHub Actions workflow runs them inside Docker.

# AutoDelete

### _retention policies for 'gamers'_

**AutoDelete** is a Discord bot that will automatically delete messages from a designated channel.

Messages are deleted on a "rolling" basis -- if you set a 24-hour live time, each message will be deleted 24 hours after it is posted (as opposed to all messages being deleted every 24 hours).

If you have an urgent message about the operation of the bot, say `@AutoDelete adminhelp ... your message here ...` and I'll get back to you as soon as I see it.

Add it to your server here: https://autodelete.riking.org/discord_auto_delete/oauth/start

**[Support me on Patreon](https://patreon.com/riking)** if you enjoy the bot or to help keep it running! https://www.patreon.com/riking

Announcements server: https://discord.gg/FUGn8yE

## Usage

Create a new "purged" channel where messages will automatically be deleted. Someone with MANAGE_MESSAGES permission (usually an admin) needs to say `@AutoDelete start 100 24h` to start the bot and tell it which channel you are using.

If slash commands are registered for the bot, the same channel can also be
configured with `/autodelete set duration:24h count:100`. The original mention
commands are still supported.

The `100` in the start command is the maximum number of live messages in the channel before the oldest is deleted.
The `24h` is a duration after which every message will be deleted. [Acceptable units](https://godoc.org/time#ParseDuration) are `h` for hours, `m` for minutes, `s` for seconds. *Warning*: Durations of a day or longer still need to be specified in hours.

Make sure to mention the **bot user** and not the role alias!

![Select the mention option with #6949 on the end.](docs/mention-user-not-role.png)

A "voice-text" channel might want a shorter duration, e.g. 30m or 10m, when you just want "immediate" chat with no memory.

*The bot must have permission to read (obviously) and send messages in the channel you are using*, in addition to the Manage Messages permission. If the bot is missing permissions, it will disable itself and attempt to tell you, though this usually won't work when it can't send messages.

To turn off the bot, use `@AutoDelete set 0` to turn off auto-deletion.

For a quick reminder of these rules, just say `@AutoDelete help`.

If you need extra help, say `@AutoDelete adminhelp ... message ...` to send a message to the support guild.

## Deployment

### Docker

The Docker image builds the bot inside the container using the Go toolchain from
the pinned builder image. You do not need Go installed on the host.

```
docker build -t autodelete:local .
```

Or with Compose:

```
docker compose up --build -d
```

The image expects a runtime config file and writable data directory to be
mounted into `/autodelete`. For Docker, set `http.listen` in `config.yml` to
`0.0.0.0:2202` if you want the OAuth callback endpoint reachable through
`-p 2202:2202`.

Required mounts:

```
/path/to/storage/config.yml:/autodelete/config.yml
/path/to/storage/data/:/autodelete/data/
```

The container runs as UID/GID `10001`, so the mounted data directory must be
writable by that user on Linux hosts. The included `compose.yml` uses a named
volume for `/autodelete/data` to avoid host bind-mount permission issues.

Example:

```
docker run -d -p 2202:2202/tcp \
 --name Autodelete \
 -v /opt/AutoDelete/config.yml:/autodelete/config.yml \
 -v /opt/AutoDelete/data/:/autodelete/data/ \
 --restart=always \
 autodelete:local
```

### Legacy Files

Original operator-specific setup scripts, Caddy examples, Prometheus examples,
and systemd unit files have been moved to `docs/legacy/`. They are retained for
reference only. Docker and Compose are the maintained self-hosting path for this
checkout.

## Policy

The following two sections apply only to the hosted, community instance that can be invited to your server at the link above, as well as the help server and this GitHub repository.

Any changes to the following policies will be announced on the support server in the #announce channel.

### Privacy

_The following section is a DRAFT and may be incomplete and is subject to change, though the information present is correct to the best of my knowledge._

No message content is ever retained, except in the case when a message "@-mentions" the bot, where it may be retained to provide support or improve the bot. The "adminhelp" command transmits the provided message content to a channel in Discord and is subject to Discord's retention policies. Deleting a command invocation via the Discord interface has no effect on how long the bot's information about the invocation is stored.

The "community instance" of the bot will retain operational usage data, including data that identifies a particular guild or channel ID and/or with high-resolution timestamps. The full form of this data will be retained for 45 days ([cite](docs/legacy/prometheus-autodelete-aggregator.service#L6)), and aggregated or summarized forms will be retained for up to 1.5 years. Usage data will not be used for commercial purposes except for the purpose of encouraging people to financially support the bot in a non-automated manner (in particular, usage data will not be sold or provided to any third party).

In order to faciliate product support, and response and detection of violations of the Acceptable Use Policy, an automated scan of your Guild structure will be performed and a report produced, with a focus on users and roles carrying the _Manage Messages_ permission and channels where the bot is or was active. These reports may be shared with a limited audience to the extent necessary to identify or cure violations of the Acceptable Use Policy.

Contact Riking via the announcements server if you would like to request a copy of this data under the GDPR or equivalent consumer rights legislation.

The settings for a channel are kept on disk with the channel ID, guild ID, pinned message IDs, pin version timestamp, and time / count settings together. In the case that a channel is removed from the bot, either through `set 0` or kicking the bot from the server, these settings are deleted. Backup or archival copies of the settings may be retained indefinitely but will not be used except for the purposes of disaster recovery.

### Acceptable Use

The bot may not be used to perform or to assist with any of the following actions:

 - improperly use support channels to make false reports;
 - engage in conduct that is fraudulent or illegal;
 - generally, to cover up, hide, or perform any violation of the Discord Terms of Service;
 - to cause the bot to violate the Discord Terms of Service if it would not have violated those terms without your actions;
 - strain the technical infrastructure of the bot with an unreasonable volume or rate of requests, or requests designed to create an unreasonable load, except with explicit permission to conduct a specific technical or security test;
 - any use of the bot in a role where malfunction, improper operation, or normal operation of the bot could cause damages that exceed the greater of (a) $100 US Dollars or (b) the amount you have paid to the Operator of the bot;

Violations of the Acceptable Use Policy may be dealt with in one or more of the following manners:

 - An informal warning from the Operator, sent via the help server, via Discord DM, or from the bot's account through Discord.
 - A formal warning from the Operator, sent from the bot's account through Discord.
 - Removal of service from your guild, with or without warning.
 - Refusal of service to any guilds a particular user operates or has moderation capabilities on.
 - Referral of incident details to Discord, law enforcement, or other competent authorities.
 - Cooperating with investigations by Discord, law enforcement, or other competent authorities.

While the above list of remedies is generally ordered by severity, the Operator has no obligation to respect the ordering of the list, to enact any specific remedy, or to take action against any specific violation. Lack of action in response to a violation is not a waiver against future remedial action (in particular, note limited investigational capacity).

If you cannot comply with the Acceptable Use Policy, you must download the code of the bot and run it on your own infrastructure, accepting all responsibility for your actions.

### Limitation of Liability

***Under no circumstance will Operators's total liability arising from your use of the service exceed the greater of (a) the amount of fees Operator received from you or (b) $100 US dollars. This includes consequential, incidential, special, punitive, or indirect damages, based on any legal theory, even if the side liable is advised that the other may suffer damages, and even if You paid no fees at all.*** Some jurisdictions do not allow the exclusion of implied warranties or limitation of liability for incidental or consequential damages. In these jurisdictions, Operator's liability will be limited to the greatest extent permitted by law.

The service is provided to you without obligation of payment, and it is your responsibility to take actions to account for potentially harmful actions it may perform.

As a reminder, the Apache License, Version 2, and not the above paragraphs, applies to source distributions of this software:

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
