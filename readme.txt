#install and setup golang
	install golang
#install node.js
#also install Wails CLI (v2) — Install via Go terminal command:
	go install github.com/wailsapp/wails/v2/cmd/wails@latest

#after installing go , wails and node 
	cd E2EEcloud\frontend
	npm install 
	cd ..
# in root directory (E2EEcloud) run 
	wails doctor	
	wails dev #to run the app in development mode 
