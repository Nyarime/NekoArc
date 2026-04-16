export namespace main {
	
	export class PackOptions {
	    input: string;
	    format: string;
	    level: number;
	    fec: number;
	    password: string;
	    solid: boolean;
	    sfx: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PackOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.format = source["format"];
	        this.level = source["level"];
	        this.fec = source["fec"];
	        this.password = source["password"];
	        this.solid = source["solid"];
	        this.sfx = source["sfx"];
	    }
	}
	export class Result {
	    success: boolean;
	    message: string;
	    duration: number;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.duration = source["duration"];
	    }
	}

}

