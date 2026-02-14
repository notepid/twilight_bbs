package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/notepid/twilight_bbs/internal/ansi"
	"github.com/notepid/twilight_bbs/internal/config"
	"github.com/notepid/twilight_bbs/internal/db"
	"github.com/notepid/twilight_bbs/internal/door"
	"github.com/notepid/twilight_bbs/internal/filearea"
	"github.com/notepid/twilight_bbs/internal/menu"
	"github.com/notepid/twilight_bbs/internal/message"
	"github.com/notepid/twilight_bbs/internal/node"
	"github.com/notepid/twilight_bbs/internal/server"
	"github.com/notepid/twilight_bbs/internal/terminal"
	"github.com/notepid/twilight_bbs/internal/user"
	"github.com/notepid/twilight_bbs/internal/chat"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting %s (sysop: %s)", cfg.BBS.Name, cfg.BBS.Sysop)

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.Paths.Data, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Open database
	database, err := db.Open(cfg.Paths.Database)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()
	log.Printf("Database opened: %s", cfg.Paths.Database)

	// Create repositories
	userRepo := user.NewRepo(database.DB)
	messageRepo := message.NewRepo(database.DB)
	fileRepo := filearea.NewRepo(database.DB)

	// Create chat broker
	chatBroker := chat.NewBroker()

	// Create door launcher
	doorLauncher := door.NewLauncher(
		cfg.Doors.DosemuPath,
		cfg.Doors.DriveC,
		filepath.Join(cfg.Paths.Data, "doors_tmp"),
	)

	// Create menu registry and scan for menus
	menuRegistry := menu.NewRegistry(cfg.Paths.Menus)
	if err := menuRegistry.Scan(); err != nil {
		log.Fatalf("Failed to scan menus: %v", err)
	}
	log.Printf("Loaded %d menus from %s", len(menuRegistry.List()), cfg.Paths.Menus)

	// Create ANSI display file loader
	ansiLoader := ansi.NewLoader(cfg.Paths.Menus, cfg.Paths.Text)

	// Create node manager
	nodeMgr := node.NewManager(cfg.BBS.MaxNodes, cfg.BBS.Name, cfg.BBS.Sysop)

	// handleConnection wires up a new node session from any connection type.
	handleConnection := func(term *terminal.Terminal, remoteAddr string) {
		nodeID, ok := nodeMgr.Acquire()
		if !ok {
			term.SendLn("Sorry, all nodes are busy. Please try again later.")
			term.Close()
			return
		}

		n := node.NewNode(nodeID, term, remoteAddr)
		n.MenuRegistry = menuRegistry
		n.ANSILoader = ansiLoader
		n.UserRepo = userRepo
		n.MessageRepo = messageRepo
		n.FileRepo = fileRepo
		n.ChatBroker = chatBroker
		n.DoorLauncher = doorLauncher
		n.DB = database.DB

		nodeMgr.Add(n)
		n.Run(nodeMgr)
	}

	// --- Telnet server ---
	telnetListener := server.NewListener(cfg.Server.TelnetPort, func(tc *server.TelnetConn) {
		// Negotiate telnet options (echo, SGA, NAWS, terminal type)
		if err := tc.Negotiate(); err != nil {
			log.Printf("Telnet negotiation error from %s: %v", tc.RemoteAddr(), err)
			tc.Close()
			return
		}

		term := terminal.New(tc, tc.Width, tc.Height, tc.ANSICapable)
		term.SetEchoControl(tc.SetEcho)

		handleConnection(term, tc.RemoteAddr().String())
	})

	go func() {
		if err := telnetListener.ListenAndServe(); err != nil {
			log.Fatalf("Telnet server error: %v", err)
		}
	}()

	// --- SSH server ---
	hostKeyPath := filepath.Join(cfg.Paths.Data, "ssh_host_key")
	sshListener, err := server.NewSSHListener(cfg.Server.SSHPort, hostKeyPath, func(sc *server.SSHConn, remoteAddr string) {
		term := terminal.New(sc, sc.Width, sc.Height, sc.ANSICapable)

		handleConnection(term, remoteAddr)
	})
	if err != nil {
		log.Fatalf("Failed to create SSH listener: %v", err)
	}

	go func() {
		if err := sshListener.ListenAndServe(); err != nil {
			log.Fatalf("SSH server error: %v", err)
		}
	}()

	// --- Health server ---
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	healthServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.HealthPort),
		Handler:           healthMux,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Health server error: %v", err)
		}
	}()

	// --- Graceful shutdown ---
	fmt.Printf("\n%s is running\n", cfg.BBS.Name)
	fmt.Printf("  Telnet: port %d\n", cfg.Server.TelnetPort)
	fmt.Printf("  SSH:    port %d\n", cfg.Server.SSHPort)
	fmt.Printf("  Health: port %d\n", cfg.Server.HealthPort)
	fmt.Printf("  Nodes:  0/%d\n", cfg.BBS.MaxNodes)
	fmt.Println("\nPress Ctrl+C to shut down.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	log.Printf("Received signal %v, shutting down...", sig)

	// Notify all connected nodes
	nodeMgr.Broadcast("System is shutting down NOW. Goodbye!")
	for _, n := range nodeMgr.List() {
		n.Disconnect()
	}

	log.Printf("%s shut down complete.", cfg.BBS.Name)
}
