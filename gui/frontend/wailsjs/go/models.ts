export namespace core {
	
	export class ActionResult {
	    ok: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ActionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	    }
	}
	export class RemoteState {
	    name: string;
	    localPath: string;
	    mountUnit: string;
	    syncServiceUnit: string;
	    syncTimerUnit: string;
	    mountActive: boolean;
	    mountEnabled: boolean;
	    syncActive: boolean;
	    syncTimerEnabled: boolean;
	    lastError?: string;
	
	    static createFrom(source: any = {}) {
	        return new RemoteState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.localPath = source["localPath"];
	        this.mountUnit = source["mountUnit"];
	        this.syncServiceUnit = source["syncServiceUnit"];
	        this.syncTimerUnit = source["syncTimerUnit"];
	        this.mountActive = source["mountActive"];
	        this.mountEnabled = source["mountEnabled"];
	        this.syncActive = source["syncActive"];
	        this.syncTimerEnabled = source["syncTimerEnabled"];
	        this.lastError = source["lastError"];
	    }
	}
	export class AppSnapshot {
	    trayState: string;
	    remotes: RemoteState[];
	    // Go type: time
	    checkedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new AppSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trayState = source["trayState"];
	        this.remotes = this.convertValues(source["remotes"], RemoteState);
	        this.checkedAt = this.convertValues(source["checkedAt"], null);
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

