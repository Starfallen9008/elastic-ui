export namespace agentstatus {
	
	export class Status {
	    state: string;
	    message: string;
	    version: string;
	    agentId: string;
	    componentsTotal: number;
	    componentsHealthy: number;
	    componentsDegraded: number;
	    componentsFailed: number;
	    checkedAt: string;
	    error?: string;
	    raw?: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.message = source["message"];
	        this.version = source["version"];
	        this.agentId = source["agentId"];
	        this.componentsTotal = source["componentsTotal"];
	        this.componentsHealthy = source["componentsHealthy"];
	        this.componentsDegraded = source["componentsDegraded"];
	        this.componentsFailed = source["componentsFailed"];
	        this.checkedAt = source["checkedAt"];
	        this.error = source["error"];
	        this.raw = source["raw"];
	    }
	}

}

export namespace config {
	
	export class Profile {
	    kibanaUrl: string;
	    esUrl: string;
	    authMethod: string;
	    username?: string;
	    apiKeyHint?: string;
	    skipTlsVerify: boolean;
	    space?: string;
	    agentId?: string;
	    hostName?: string;
	    agentPolicyId?: string;
	    packagePolicyId?: string;
	    policyName?: string;
	    boundAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kibanaUrl = source["kibanaUrl"];
	        this.esUrl = source["esUrl"];
	        this.authMethod = source["authMethod"];
	        this.username = source["username"];
	        this.apiKeyHint = source["apiKeyHint"];
	        this.skipTlsVerify = source["skipTlsVerify"];
	        this.space = source["space"];
	        this.agentId = source["agentId"];
	        this.hostName = source["hostName"];
	        this.agentPolicyId = source["agentPolicyId"];
	        this.packagePolicyId = source["packagePolicyId"];
	        this.policyName = source["policyName"];
	        this.boundAt = source["boundAt"];
	    }
	}
	export class Settings {
	    policySetupComplete: boolean;
	    profile: Profile;
	    shellIntegration: boolean;
	    notifications: boolean;
	    notificationMode: string;
	    notificationRevision?: number;
	    closeToTray: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.policySetupComplete = source["policySetupComplete"];
	        this.profile = this.convertValues(source["profile"], Profile);
	        this.shellIntegration = source["shellIntegration"];
	        this.notifications = source["notifications"];
	        this.notificationMode = source["notificationMode"];
	        this.notificationRevision = source["notificationRevision"];
	        this.closeToTray = source["closeToTray"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace service {
	
	export class AgentRef {
	    id: string;
	    hostName: string;
	    policyId: string;
	    policyName: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.hostName = source["hostName"];
	        this.policyId = source["policyId"];
	        this.policyName = source["policyName"];
	    }
	}
	export class AuditEntry {
	    time: string;
	    action: string;
	    target: string;
	    result: string;
	    detail?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.action = source["action"];
	        this.target = source["target"];
	        this.result = source["result"];
	        this.detail = source["detail"];
	    }
	}
	export class Check {
	    name: string;
	    ok: boolean;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new Check(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.ok = source["ok"];
	        this.detail = source["detail"];
	    }
	}
	export class ConnectionInput {
	    kibanaUrl: string;
	    esUrl: string;
	    authMethod: string;
	    username: string;
	    password: string;
	    apiKey: string;
	    skipTlsVerify: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kibanaUrl = source["kibanaUrl"];
	        this.esUrl = source["esUrl"];
	        this.authMethod = source["authMethod"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.apiKey = source["apiKey"];
	        this.skipTlsVerify = source["skipTlsVerify"];
	    }
	}
	export class ConnectionResult {
	    ok: boolean;
	    checks: Check[];
	    agent?: AgentRef;
	    version?: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.checks = this.convertValues(source["checks"], Check);
	        this.agent = this.convertValues(source["agent"], AgentRef);
	        this.version = source["version"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EventItem {
	    id: string;
	    time: string;
	    category: string;
	    action: string;
	    processName: string;
	    processPid: number;
	    processPath: string;
	    filePath: string;
	    destIp: string;
	    destPort: number;
	    protocol: string;
	    user: string;
	    message: string;
	    raw: string;
	
	    static createFrom(source: any = {}) {
	        return new EventItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.time = source["time"];
	        this.category = source["category"];
	        this.action = source["action"];
	        this.processName = source["processName"];
	        this.processPid = source["processPid"];
	        this.processPath = source["processPath"];
	        this.filePath = source["filePath"];
	        this.destIp = source["destIp"];
	        this.destPort = source["destPort"];
	        this.protocol = source["protocol"];
	        this.user = source["user"];
	        this.message = source["message"];
	        this.raw = source["raw"];
	    }
	}
	export class EventPage {
	    items: EventItem[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new EventPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], EventItem);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class IndependentPolicyState {
	    independent: boolean;
	    applied: boolean;
	    name: string;
	    policyId: string;
	    packageId: string;
	    hostCount: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new IndependentPolicyState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.independent = source["independent"];
	        this.applied = source["applied"];
	        this.name = source["name"];
	        this.policyId = source["policyId"];
	        this.packageId = source["packageId"];
	        this.hostCount = source["hostCount"];
	        this.message = source["message"];
	    }
	}
	export class NotificationState {
	    mode: string;
	    ready: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new NotificationState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.ready = source["ready"];
	        this.message = source["message"];
	    }
	}
	export class ProcessNode {
	    workingDir: string;
	    entityId: string;
	    name: string;
	    pid: number;
	    parentEntityId: string;
	    executable: string;
	    commandLine: string;
	    args: string[];
	    user: string;
	    timestamp: string;
	    start: string;
	    end: string;
	    eventAction: string;
	    resolved: boolean;
	    isAlert: boolean;
	    depth: number;
	
	    static createFrom(source: any = {}) {
	        return new ProcessNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workingDir = source["workingDir"];
	        this.entityId = source["entityId"];
	        this.name = source["name"];
	        this.pid = source["pid"];
	        this.parentEntityId = source["parentEntityId"];
	        this.executable = source["executable"];
	        this.commandLine = source["commandLine"];
	        this.args = source["args"];
	        this.user = source["user"];
	        this.timestamp = source["timestamp"];
	        this.start = source["start"];
	        this.end = source["end"];
	        this.eventAction = source["eventAction"];
	        this.resolved = source["resolved"];
	        this.isAlert = source["isAlert"];
	        this.depth = source["depth"];
	    }
	}
	export class ProcessChain {
	    notice: string;
	    alertId: string;
	    alertName: string;
	    alertTime: string;
	    hostName: string;
	    nodes: ProcessNode[];
	    children: ProcessNode[];
	
	    static createFrom(source: any = {}) {
	        return new ProcessChain(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.notice = source["notice"];
	        this.alertId = source["alertId"];
	        this.alertName = source["alertName"];
	        this.alertTime = source["alertTime"];
	        this.hostName = source["hostName"];
	        this.nodes = this.convertValues(source["nodes"], ProcessNode);
	        this.children = this.convertValues(source["children"], ProcessNode);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProcessInfo {
	    pid: number;
	    entityId: string;
	    user: string;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new ProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.entityId = source["entityId"];
	        this.user = source["user"];
	        this.command = source["command"];
	    }
	}
	
	export class Protection {
	    malware: string;
	    ransomware: string;
	    memory: string;
	    behavior: string;
	    deviceControl: string;
	    avRegistration: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Protection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.malware = source["malware"];
	        this.ransomware = source["ransomware"];
	        this.memory = source["memory"];
	        this.behavior = source["behavior"];
	        this.deviceControl = source["deviceControl"];
	        this.avRegistration = source["avRegistration"];
	    }
	}
	export class QuarantineOperation {
	    id: string;
	    path: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new QuarantineOperation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.error = source["error"];
	    }
	}
	export class Response {
	    action: string;
	    message: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new Response(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.message = source["message"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class RetrievedFile {
	    path: string;
	    fileName: string;
	    size: number;
	    sha256: string;
	    savedPath: string;
	    zipPath: string;
	    savedDir: string;
	    downloadUri: string;
	    isDirectory: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RetrievedFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.fileName = source["fileName"];
	        this.size = source["size"];
	        this.sha256 = source["sha256"];
	        this.savedPath = source["savedPath"];
	        this.zipPath = source["zipPath"];
	        this.savedDir = source["savedDir"];
	        this.downloadUri = source["downloadUri"];
	        this.isDirectory = source["isDirectory"];
	    }
	}
	export class ScanFile {
	    id: string;
	    time: string;
	    path: string;
	    malwareName: string;
	    sha256: string;
	    outcome: string;
	    detail: string;
	    quarantinePath: string;
	    recoverable: boolean;
	    availability: string;
	    recovery: string;
	    savedPath: string;
	    recoveredAt: string;
	    recoveryError: string;
	    trustStatus: string;
	
	    static createFrom(source: any = {}) {
	        return new ScanFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.time = source["time"];
	        this.path = source["path"];
	        this.malwareName = source["malwareName"];
	        this.sha256 = source["sha256"];
	        this.outcome = source["outcome"];
	        this.detail = source["detail"];
	        this.quarantinePath = source["quarantinePath"];
	        this.recoverable = source["recoverable"];
	        this.availability = source["availability"];
	        this.recovery = source["recovery"];
	        this.savedPath = source["savedPath"];
	        this.recoveredAt = source["recoveredAt"];
	        this.recoveryError = source["recoveryError"];
	        this.trustStatus = source["trustStatus"];
	    }
	}
	export class ScanResults {
	    items: ScanFile[];
	    total: number;
	    notice: string;
	
	    static createFrom(source: any = {}) {
	        return new ScanResults(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], ScanFile);
	        this.total = source["total"];
	        this.notice = source["notice"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ScanTask {
	    id: string;
	    actionId: string;
	    path: string;
	    status: string;
	    startedAt: string;
	    completedAt?: string;
	    detail?: string;
	    found: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.actionId = source["actionId"];
	        this.path = source["path"];
	        this.status = source["status"];
	        this.startedAt = source["startedAt"];
	        this.completedAt = source["completedAt"];
	        this.detail = source["detail"];
	        this.found = source["found"];
	    }
	}
	export class StatusView {
	    connected: boolean;
	    connectionError: string;
	    protectionError: string;
	    kibanaVersion: string;
	    license: string;
	    hostName: string;
	    agentId: string;
	    agentStatus: string;
	    lastCheckin: string;
	    endpointStatus: string;
	    isolation: boolean;
	    tamperProtection: boolean;
	    agentVersion: string;
	    endpointVersion: string;
	    policyName: string;
	    policyVersion: string;
	    endpointPolicyVersion: string;
	    protection: Protection;
	    alerts24h: number;
	    blocked24h: number;
	    lastScan?: ScanTask;
	    checks: Check[];
	    refreshAt: string;
	    scanCount: number;
	    isolationActionId?: string;
	    isolationPending: boolean;
	    packagePolicyId: string;
	    capabilities: string[];
	    protectionUpdateNote: string;
	    protectionUpdateVersion: string;
	    protectionUpdateAge: number;
	    agentRuntime: agentstatus.Status;
	
	    static createFrom(source: any = {}) {
	        return new StatusView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.connectionError = source["connectionError"];
	        this.protectionError = source["protectionError"];
	        this.kibanaVersion = source["kibanaVersion"];
	        this.license = source["license"];
	        this.hostName = source["hostName"];
	        this.agentId = source["agentId"];
	        this.agentStatus = source["agentStatus"];
	        this.lastCheckin = source["lastCheckin"];
	        this.endpointStatus = source["endpointStatus"];
	        this.isolation = source["isolation"];
	        this.tamperProtection = source["tamperProtection"];
	        this.agentVersion = source["agentVersion"];
	        this.endpointVersion = source["endpointVersion"];
	        this.policyName = source["policyName"];
	        this.policyVersion = source["policyVersion"];
	        this.endpointPolicyVersion = source["endpointPolicyVersion"];
	        this.protection = this.convertValues(source["protection"], Protection);
	        this.alerts24h = source["alerts24h"];
	        this.blocked24h = source["blocked24h"];
	        this.lastScan = this.convertValues(source["lastScan"], ScanTask);
	        this.checks = this.convertValues(source["checks"], Check);
	        this.refreshAt = source["refreshAt"];
	        this.scanCount = source["scanCount"];
	        this.isolationActionId = source["isolationActionId"];
	        this.isolationPending = source["isolationPending"];
	        this.packagePolicyId = source["packagePolicyId"];
	        this.capabilities = source["capabilities"];
	        this.protectionUpdateNote = source["protectionUpdateNote"];
	        this.protectionUpdateVersion = source["protectionUpdateVersion"];
	        this.protectionUpdateAge = source["protectionUpdateAge"];
	        this.agentRuntime = this.convertValues(source["agentRuntime"], agentstatus.Status);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Threat {
	    id: string;
	    time: string;
	    message: string;
	    ruleName: string;
	    severity: number;
	    riskScore: number;
	    blocked: boolean;
	    action: string;
	    processName: string;
	    processPath: string;
	    filePath: string;
	    malwareName: string;
	    sha256: string;
	    read: boolean;
	    category: string;
	    detectionType: string;
	    responseTime: string;
	
	    static createFrom(source: any = {}) {
	        return new Threat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.time = source["time"];
	        this.message = source["message"];
	        this.ruleName = source["ruleName"];
	        this.severity = source["severity"];
	        this.riskScore = source["riskScore"];
	        this.blocked = source["blocked"];
	        this.action = source["action"];
	        this.processName = source["processName"];
	        this.processPath = source["processPath"];
	        this.filePath = source["filePath"];
	        this.malwareName = source["malwareName"];
	        this.sha256 = source["sha256"];
	        this.read = source["read"];
	        this.category = source["category"];
	        this.detectionType = source["detectionType"];
	        this.responseTime = source["responseTime"];
	    }
	}
	export class ThreatDetail {
	    id: string;
	    time: string;
	    message: string;
	    ruleName: string;
	    severity: number;
	    riskScore: number;
	    blocked: boolean;
	    action: string;
	    processName: string;
	    processPath: string;
	    filePath: string;
	    malwareName: string;
	    sha256: string;
	    read: boolean;
	    category: string;
	    detectionType: string;
	    responseTime: string;
	    quarantineFile?: ScanFile;
	    commandLine: string;
	    workingDir: string;
	    parentName: string;
	    parentPath: string;
	    parentCmd: string;
	    user: string;
	    responses: Response[];
	    tactics: string[];
	    techniques: string[];
	    hostOs: string;
	
	    static createFrom(source: any = {}) {
	        return new ThreatDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.time = source["time"];
	        this.message = source["message"];
	        this.ruleName = source["ruleName"];
	        this.severity = source["severity"];
	        this.riskScore = source["riskScore"];
	        this.blocked = source["blocked"];
	        this.action = source["action"];
	        this.processName = source["processName"];
	        this.processPath = source["processPath"];
	        this.filePath = source["filePath"];
	        this.malwareName = source["malwareName"];
	        this.sha256 = source["sha256"];
	        this.read = source["read"];
	        this.category = source["category"];
	        this.detectionType = source["detectionType"];
	        this.responseTime = source["responseTime"];
	        this.quarantineFile = this.convertValues(source["quarantineFile"], ScanFile);
	        this.commandLine = source["commandLine"];
	        this.workingDir = source["workingDir"];
	        this.parentName = source["parentName"];
	        this.parentPath = source["parentPath"];
	        this.parentCmd = source["parentCmd"];
	        this.user = source["user"];
	        this.responses = this.convertValues(source["responses"], Response);
	        this.tactics = source["tactics"];
	        this.techniques = source["techniques"];
	        this.hostOs = source["hostOs"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ThreatPage {
	    items: Threat[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new ThreatPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], Threat);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TrustInput {
	    kind: string;
	    value: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new TrustInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.value = source["value"];
	        this.description = source["description"];
	    }
	}
	export class TrustItem {
	    id: string;
	    itemId: string;
	    name: string;
	    description: string;
	    kind: string;
	    value: string;
	    os: string;
	    scope: string;
	    readOnly: boolean;
	    createdBy: string;
	    updatedAt: string;
	    matchKey: string;
	
	    static createFrom(source: any = {}) {
	        return new TrustItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.itemId = source["itemId"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.kind = source["kind"];
	        this.value = source["value"];
	        this.os = source["os"];
	        this.scope = source["scope"];
	        this.readOnly = source["readOnly"];
	        this.createdBy = source["createdBy"];
	        this.updatedAt = source["updatedAt"];
	        this.matchKey = source["matchKey"];
	    }
	}

}

export namespace update {
	
	export class Release {
	    version: string;
	    available: boolean;
	    url: string;
	    downloadUrl: string;
	    checksumUrl: string;
	    notes: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new Release(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.available = source["available"];
	        this.url = source["url"];
	        this.downloadUrl = source["downloadUrl"];
	        this.checksumUrl = source["checksumUrl"];
	        this.notes = source["notes"];
	        this.size = source["size"];
	    }
	}

}

