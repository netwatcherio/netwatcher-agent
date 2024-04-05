!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "nsDialogs.nsh"
!include "FileFunc.nsh"
!include "WinMessages.nsh"

Var Dialog
Var HostInput
Var HostWSInput
Var IDInput
Var PINInput

;--------------------------------
; General Attributes

Name "NetWatcher Setup"
OutFile "NetWatcherSetup.exe"
InstallDir "$PROGRAMFILES\NetWatcher"
RequestExecutionLevel admin
ShowInstDetails show
ShowUnInstDetails show

;--------------------------------
; Pages

Page directory
!insertmacro MUI_PAGE_INSTFILES
Page custom MyCustomPageCreate MyCustomPageLeave
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

;--------------------------------
; Custom Page Functions

Function MyCustomPageCreate
    nsDialogs::Create 1018
    Pop $Dialog

    ${If} $Dialog == error
        Abort
    ${EndIf}

    ; Create input fields
    nsDialogs::CreateControl "Static" ${WS_VISIBLE}|${WS_CHILD}|${SS_CENTER} 0 5u 100% 12u $Dialog "Host URL:"
    Pop $0
    nsDialogs::CreateControl "Edit" ${WS_VISIBLE}|${WS_BORDER}|${WS_TABSTOP} 0 20u 100% 12u $Dialog ""
    Pop $HostInput

    nsDialogs::CreateControl "Static" ${WS_VISIBLE}|${WS_CHILD}|${SS_CENTER} 0 40u 100% 12u $Dialog "Host WS URL:"
    Pop $0
    nsDialogs::CreateControl "Edit" ${WS_VISIBLE}|${WS_BORDER}|${WS_TABSTOP} 0 55u 100% 12u $Dialog ""
    Pop $HostWSInput

    nsDialogs::CreateControl "Static" ${WS_VISIBLE}|${WS_CHILD}|${SS_CENTER} 0 75u 100% 12u $Dialog "ID:"
    Pop $0
    nsDialogs::CreateControl "Edit" ${WS_VISIBLE}|${WS_BORDER}|${WS_TABSTOP} 0 90u 100% 12u $Dialog ""
    Pop $IDInput

    nsDialogs::CreateControl "Static" ${WS_VISIBLE}|${WS_CHILD}|${SS_CENTER} 0 110u 100% 12u $Dialog "PIN:"
    Pop $0
    nsDialogs::CreateControl "Edit" ${WS_VISIBLE}|${WS_BORDER}|${WS_TABSTOP} 0 125u 100% 12u $Dialog ""
    Pop $PINInput

    ; Set default values
    nsDialogs::SetControlText $HostInput "https://api.netwatcher.io"
    nsDialogs::SetControlText $HostWSInput "wss://api.netwatcher.io/agent_ws"
    nsDialogs::SetControlText $IDInput "65e75301b3bbdbc7126ebe97"
    nsDialogs::SetControlText $PINInput "946215265"

    nsDialogs::Show
FunctionEnd

Function MyCustomPageLeave
    ; Retrieve input values
    nsDialogs::GetControlText $HostInput $0
    nsDialogs::GetControlText $HostWSInput $1
    nsDialogs::GetControlText $IDInput $2
    nsDialogs::GetControlText $PINInput $3

    ; Write the configuration to a file
    Push $0
    Push $1
    Push $2
    Push $3
    Call WriteConfig
FunctionEnd

Function WriteConfig
    Exch $3
    Exch
    Exch $2
    Exch 2
    Exch $1
    Exch 3
    Pop $0

    ; Open the config file for writing
    FileOpen $0 "$INSTDIR\config.conf" w

    ; Write each piece of information
    FileWrite $0 "HOST=$1$\r$\n"
    FileWrite $0 "HOST_WS=$2$\r$\n"
    FileWrite $0 "ID=$3$\r$\n"
    FileWrite $0 "PIN=$4$\r$\n"

    ; Close the file
    FileClose $0
FunctionEnd

;--------------------------------
; Installation Section

Section "Install"
    SetOutPath "$INSTDIR"

    ; Install the executable and lib folder
    File ".\netwatcher-agent.exe"
    File /r ".\lib\*.exe"

    ; Create service
    nsExec::ExecToLog 'sc create "NetWatcherAgent" binPath= "$INSTDIR\netwatcher-agent.exe"'
    nsExec::ExecToLog 'sc start "NetWatcherAgent"'
SectionEnd

;--------------------------------
; Uninstallation Section

Section "Uninstall"
    ; Stop and remove the service
    nsExec::ExecToLog 'sc stop "NetWatcherAgent"'
    Sleep 3000 ; Wait for the service to stop
    nsExec::ExecToLog 'sc delete "NetWatcherAgent"'

    ; Remove installed files and directories
    Delete "$INSTDIR\netwatcher-agent.exe"
    RMDir /r "$INSTDIR\lib"
    Delete "$INSTDIR\config.conf"

    ; Remove installation directory
    RMDir "$INSTDIR"
SectionEnd