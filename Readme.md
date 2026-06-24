Kuafu is chasing the sun! Make it great here. 



now we start a project, this is kuafu. we firstly come out some design, such as what we want to build, arechitecture,language etc. it should be a md file. this will be co-authored by all the team. PM should coordinate the work. the project is setup to build a word-class GPU management system, which will be k8s based, support UI and commandline interface like lsf/pbs or slurm. it will have different command groups such as resource managment, job management and so on. there is a sample project in q:\\gitroot\\pai. the job scheduling, which is actually a launcher to do parallel jobmanamgene. we even could use directly rather than develop by ourselve. by the way, pai is very well organized and we could leverage. the jobs had better be manged by a mongodb database to make it easy, then a problem is how to manage the data in database or etcd. should be balanced.  A vision is that the platform could manage all the gpus up to 4000 with different skus such as a100, mi300 etc. a user could submit a job to a queue or login and reserve a gpu by paying.  now PM coodinate the work. research should be able to answer all questions. 

