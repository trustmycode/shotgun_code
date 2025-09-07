export namespace main {
	
	export class GeminiPart {
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new GeminiPart(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	    }
	}
	export class GeminiMessage {
	    role: string;
	    parts: GeminiPart[];
	
	    static createFrom(source: any = {}) {
	        return new GeminiMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.parts = this.convertValues(source["parts"], GeminiPart);
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
	export class ChatRequest {
	    history: GeminiMessage[];
	    message: string;
	    temperature: number;
	
	    static createFrom(source: any = {}) {
	        return new ChatRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.history = this.convertValues(source["history"], GeminiMessage);
	        this.message = source["message"];
	        this.temperature = source["temperature"];
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
	export class FileNode {
	    name: string;
	    path: string;
	    relPath: string;
	    isDir: boolean;
	    children?: FileNode[];
	    isGitignored: boolean;
	    isCustomIgnored: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.relPath = source["relPath"];
	        this.isDir = source["isDir"];
	        this.children = this.convertValues(source["children"], FileNode);
	        this.isGitignored = source["isGitignored"];
	        this.isCustomIgnored = source["isCustomIgnored"];
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

