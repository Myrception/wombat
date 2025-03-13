// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT
import {app} from '../models';
import {context} from '../models';

export function Cancel():Promise<void>;

export function CloseSend():Promise<void>;

export function Connect(arg1:any,arg2:any,arg3:boolean):Promise<void>;

export function DeleteWorkspace(arg1:string):Promise<void>;

export function ExportCommands(arg1:string,arg2:string,arg3:any):Promise<app.commands>;

export function FindProtoFiles():Promise<Array<string>>;

export function GetMetadata(arg1:string):Promise<app.headers>;

export function GetRawMessageState(arg1:string):Promise<string>;

export function GetReflectMetadata(arg1:string):Promise<app.headers>;

export function GetWindowInfo():Promise<Record<string, any>>;

export function GetWorkspaceOptions():Promise<app.options>;

export function ImportCommand(arg1:string,arg2:string):Promise<void>;

export function ListWorkspaces():Promise<Array<app.options>>;

export function RetryConnection():Promise<void>;

export function SelectDirectory():Promise<string>;

export function SelectMethod(arg1:string,arg2:string,arg3:any):Promise<void>;

export function SelectWorkspace(arg1:string):Promise<void>;

export function Send(arg1:string,arg2:string,arg3:any):Promise<void>;

export function Shutdown(arg1:context.Context):Promise<void>;

export function WailsShutdown():Promise<void>;
