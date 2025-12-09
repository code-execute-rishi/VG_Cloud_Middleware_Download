'use client';

import { useEffect, useRef, useState } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { Room, RoomEvent, RemoteDataChannel } from 'livekit-client';
import '@xterm/xterm/css/xterm.css';

export function SSHTerminal({ room, className = '' }) {
  const terminalRef = useRef(null);
  const terminal = useRef(null);
  const fitAddon = useRef(null);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    if (!room) return;

    terminal.current = new Terminal({
      cursorBlink: true,
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
        cursor: '#aeafad',
        black: '#000000',
        red: '#cd3131',
        green: '#0dbc79',
        yellow: '#e5e510',
        blue: '#2472c8',
        magenta: '#bc3fbc',
        cyan: '#11a8cd',
        white: '#e5e5e5',
        brightBlack: '#666666',
        brightRed: '#f14c4c',
        brightGreen: '#23d18b',
        brightYellow: '#f5f543',
        brightBlue: '#3b8eea',
        brightMagenta: '#d670d6',
        brightCyan: '#29b8db',
        brightWhite: '#e5e5e5',
      },
      fontSize: 14,
      fontFamily: 'Consolas, "Courier New", monospace',
    });

    fitAddon.current = new FitAddon();
    terminal.current.loadAddon(fitAddon.current);

    if (terminalRef.current) {
      terminal.current.open(terminalRef.current);
      fitAddon.current.fit();

      terminal.current.writeln('Connecting to drone via SSH...\r\n');
    }

    const handleDataReceived = (payload, participant, kind, topic) => {
      if (kind === 'reliable' && topic === 'ssh') {
        const data = new TextDecoder().decode(payload);
        if (terminal.current) {
          terminal.current.write(data);
          setIsConnected(true);
        }
      }
    };

    const handleConnected = () => {
      if (terminal.current) {
        terminal.current.writeln('\r\n✅ Connected to drone terminal\r\n');
        setIsConnected(true);

        terminal.current.onData((data) => {
          if (room) {
            const encoder = new TextEncoder();
            const buffer = encoder.encode(data);
            room.localParticipant.publishData(buffer, {
              kind: 'reliable',
              topic: 'ssh',
            });
          }
        });
      }
    };

    if (room.state === 'connected') {
      handleConnected();
    }

    room.on(RoomEvent.Connected, handleConnected);
    room.on(RoomEvent.DataReceived, handleDataReceived);

    const handleResize = () => {
      if (fitAddon.current && terminal.current) {
        fitAddon.current.fit();
      }
    };

    window.addEventListener('resize', handleResize);

    return () => {
      room.off(RoomEvent.Connected, handleConnected);
      room.off(RoomEvent.DataReceived, handleDataReceived);
      window.removeEventListener('resize', handleResize);
      if (terminal.current) {
        terminal.current.dispose();
      }
    };
  }, [room, isConnected]);

  return (
    <div className={`w-full h-full bg-[#1e1e1e] rounded-lg overflow-hidden ${className}`}>
      <div ref={terminalRef} className="w-full h-full" />
    </div>
  );
}

