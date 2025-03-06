export namespace app {
	
	export class commands {
	    grpcurl: string;
	
	    static createFrom(source: any = {}) {
	        return new commands(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.grpcurl = source["grpcurl"];
	    }
	}
	export class header {
	    key: string;
	    val: string;
	
	    static createFrom(source: any = {}) {
	        return new header(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.val = source["val"];
	    }
	}
	export class protos {
	    files: string[];
	    roots: string[];
	
	    static createFrom(source: any = {}) {
	        return new protos(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.files = source["files"];
	        this.roots = source["roots"];
	    }
	}
	export class options {
	    id: string;
	    addr: string;
	    reflect: boolean;
	    protos: protos;
	    insecure: boolean;
	    plaintext: boolean;
	    rootca: string;
	    clientcert: string;
	    clientkey: string;
	
	    static createFrom(source: any = {}) {
	        return new options(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.addr = source["addr"];
	        this.reflect = source["reflect"];
	        this.protos = this.convertValues(source["protos"], protos);
	        this.insecure = source["insecure"];
	        this.plaintext = source["plaintext"];
	        this.rootca = source["rootca"];
	        this.clientcert = source["clientcert"];
	        this.clientkey = source["clientkey"];
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

