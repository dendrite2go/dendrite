FROM lnl7/nix

RUN nix-channel --add https://nixos.org/channels/nixos-unstable
RUN nix-channel --update

ENV NIX_PATH='/nix/var/nix/profiles/per-user/root/channels/nixos'
RUN rm -f '/root/.nix-defexpr/nixos'

RUN nix-env -i bash-interactive
RUN ln -s /root/.nix-profile/bin/bash /bin/.

RUN nix-env -f '<.>' -iA curl
RUN nix-env -f '<.>' -iA gnumake
RUN nix-env -f '<.>' -iA go
RUN nix-env -f '<.>' -iA vgo2nix
RUN nix-env -f '<.>' -iA protobuf

RUN mkdir -p /root/.config/nixpkgs ; echo '{ allowBroken = true; }' > /root/.config/nixpkgs/config.nix

RUN mkdir /src
RUN cd /src ; git clone https://github.com/AxonIQ/axon-server-api.git
